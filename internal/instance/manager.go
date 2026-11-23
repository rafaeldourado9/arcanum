// Package instance gerencia multiplas conexoes WhatsApp simultaneas.
// Cada instancia possui seu proprio provider (whatsmeow), rate limiter,
// webhook forwarder, e configuracoes independentes, gravadas em <nome>.json para
// sobreviver a reinicios.
// Thread-safe: o mapa de instancias tem o lock do Manager e a configuracao de cada
// instancia tem o lock da propria instancia.
package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/antiban"
	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/provider"
	"github.com/rafaeldourado9/arcanum/internal/webhook"
)

type Instance struct {
	Name      string
	Provider  provider.WhatsAppProvider
	Limiter   *antiban.RateLimiter
	CreatedAt time.Time

	mu         sync.RWMutex
	settings   Settings
	webhookCfg WebhookConfig
	forwarder  *webhook.Forwarder
}

type WebhookConfig struct {
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled bool     `json:"enabled"`
}

type Settings struct {
	RejectCalls     bool `json:"rejectCalls"`
	GroupsIgnore    bool `json:"groupsIgnore"`
	AlwaysOnline    bool `json:"alwaysOnline"`
	ReadMessages    bool `json:"readMessages"`
	ReadStatus      bool `json:"readStatus"`
	SyncFullHistory bool `json:"syncFullHistory"`
}

type InstanceInfo struct {
	Name      string                    `json:"instanceName"`
	Status    provider.ConnectionStatus `json:"status"`
	Phone     string                    `json:"phone,omitempty"`
	CreatedAt time.Time                 `json:"createdAt"`
	Settings  Settings                  `json:"settings"`
	Webhook   WebhookConfig             `json:"webhook"`
}

// Settings devolve uma copia das configuracoes (seguro para ler sem lock).
func (i *Instance) Settings() Settings {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.settings
}

// WebhookConfig devolve uma copia da configuracao de webhook.
func (i *Instance) WebhookConfig() WebhookConfig {
	i.mu.RLock()
	defer i.mu.RUnlock()
	c := i.webhookCfg
	c.Events = append([]string(nil), i.webhookCfg.Events...)
	return c
}

// Forwarder e o cliente de webhook atual da instancia.
func (i *Instance) Forwarder() *webhook.Forwarder {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.forwarder
}

// webhookOn diz se ha webhook ligado, e devolve o forwarder para usar.
func (i *Instance) webhookOn() (*webhook.Forwarder, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.forwarder, i.webhookCfg.Enabled && i.webhookCfg.URL != ""
}

// ReportSendFailure avisa o webhook (se ligado) que um envio foi abandonado.
func (i *Instance) ReportSendFailure(to, text string) {
	if fwd, on := i.webhookOn(); on {
		fwd.ReportSendFailure(to, text)
	}
}

func (i *Instance) info() InstanceInfo {
	return InstanceInfo{
		Name:      i.Name,
		Status:    i.Provider.Status(),
		Phone:     i.Provider.PhoneNumber(),
		CreatedAt: i.CreatedAt,
		Settings:  i.Settings(),
		Webhook:   i.WebhookConfig(),
	}
}

// persisted e o que vai para <nome>.json: o necessario para recriar a instancia igual
// depois de um reinicio. A sessao do WhatsApp em si fica no <nome>.db.
type persisted struct {
	CreatedAt time.Time     `json:"createdAt"`
	Settings  Settings      `json:"settings"`
	Webhook   WebhookConfig `json:"webhook"`
}

type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	cfg       *config.Config
	// newProvider cria o provider de cada instancia (troca em teste).
	newProvider func(dbPath string) provider.WhatsAppProvider
}

func NewManager(cfg *config.Config) *Manager {
	m := newManager(cfg, func(dbPath string) provider.WhatsAppProvider { return provider.NewWhatsmeow(dbPath) })
	m.restorePersisted()
	return m
}

func newManager(cfg *config.Config, newProvider func(string) provider.WhatsAppProvider) *Manager {
	return &Manager{instances: make(map[string]*Instance), cfg: cfg, newProvider: newProvider}
}

func (m *Manager) dbFile(name string) string   { return filepath.Join(m.cfg.DBPath, name+".db") }
func (m *Manager) metaFile(name string) string { return filepath.Join(m.cfg.DBPath, name+".json") }

// restorePersisted recria no boot cada instancia que tem <nome>.db (sessao pareada) ou
// <nome>.json (configuracao), com a configuracao gravada, e reconecta as pareadas.
// Assim um restart do container NAO derruba as instancias nem perde o webhook de cada uma.
func (m *Manager) restorePersisted() {
	entries, err := os.ReadDir(m.cfg.DBPath)
	if err != nil {
		log.Printf("[manager] cannot read db path %q: %v", m.cfg.DBPath, err)
		return
	}
	names := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		for _, ext := range []string{".db", ".json"} {
			if strings.HasSuffix(e.Name(), ext) {
				names[strings.TrimSuffix(e.Name(), ext)] = true
			}
		}
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		meta := m.loadMeta(name)
		if _, err := m.create(name, meta, false); err != nil {
			log.Printf("[manager] restore '%s' failed: %v", name, err)
			continue
		}
		log.Printf("[manager] restoring instance '%s' from disk", name)
		if _, err := os.Stat(m.dbFile(name)); err != nil {
			continue // nunca pareada: fica esperando o "conectar"
		}
		go func(n string) {
			if _, err := m.Connect(n); err != nil {
				log.Printf("[manager] reconnect '%s' failed: %v", n, err)
			}
		}(name)
	}
}

func (m *Manager) loadMeta(name string) *persisted {
	b, err := os.ReadFile(m.metaFile(name))
	if err != nil {
		return nil
	}
	var p persisted
	if err := json.Unmarshal(b, &p); err != nil {
		log.Printf("[manager] ignoring corrupt %s.json: %v", name, err)
		return nil
	}
	return &p
}

// save grava a configuracao da instancia (escrita atomica: arquivo temporario + rename,
// para uma queda no meio nao deixar JSON pela metade).
func (m *Manager) save(inst *Instance) {
	p := persisted{CreatedAt: inst.CreatedAt, Settings: inst.Settings(), Webhook: inst.WebhookConfig()}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Printf("[manager] save '%s': %v", inst.Name, err)
		return
	}
	path := m.metaFile(inst.Name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		log.Printf("[manager] save '%s': %v", inst.Name, err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("[manager] save '%s': %v", inst.Name, err)
	}
}

// ErrExists: ja ha instancia com esse nome (a API responde 409; erro de validacao e 400).
var ErrExists = errors.New("already exists")

// ErrNotFound: nao ha instancia com esse nome (a API responde 404).
var ErrNotFound = errors.New("not found")

// Create cria uma instancia nova pela API e grava a configuracao.
func (m *Manager) Create(name string, webhookURL string, events []string) (*Instance, error) {
	if webhookURL != "" {
		if err := webhook.ValidateURL(webhookURL); err != nil {
			return nil, err
		}
	}
	meta := &persisted{
		CreatedAt: time.Now().UTC(),
		Settings:  Settings{ReadMessages: true},
		Webhook:   WebhookConfig{URL: webhookURL, Events: events, Enabled: webhookURL != ""},
	}
	return m.create(name, meta, true)
}

func (m *Manager) create(name string, meta *persisted, persist bool) (*Instance, error) {
	if !ValidName(name) {
		return nil, fmt.Errorf("invalid instance name %q: use 1-64 letters, digits, '-' or '_'", name)
	}

	m.mu.Lock()
	if _, exists := m.instances[name]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("instance '%s': %w", name, ErrExists)
	}

	if meta == nil {
		// Sessao antiga, de antes do <nome>.json: valores padrao.
		meta = &persisted{CreatedAt: time.Now().UTC(), Settings: Settings{ReadMessages: true}}
	}
	whCfg := meta.Webhook
	if whCfg.URL == "" {
		// {instance} na URL global identifica a instancia (o tenant) no backend,
		// inclusive para instancias restauradas do disco apos um restart.
		whCfg.URL = strings.ReplaceAll(m.cfg.WebhookForwardURL, "{instance}", name)
		whCfg.Enabled = m.cfg.WebhookForwardURL != ""
	}
	if len(whCfg.Events) == 0 {
		whCfg.Events = []string{"messages", "status"}
	}

	wp := m.newProvider(m.dbFile(name))
	inst := &Instance{
		Name:       name,
		Provider:   wp,
		Limiter:    antiban.NewRateLimiter(m.cfg.RateLimitPerMin),
		CreatedAt:  meta.CreatedAt,
		settings:   meta.Settings,
		webhookCfg: whCfg,
		forwarder:  webhook.NewForwarder(whCfg.URL, m.cfg.WebhookFormat, m.cfg.WebhookSecret),
	}

	wp.OnMessage(func(msg provider.IncomingMessage) {
		if inst.Settings().GroupsIgnore && isGroup(msg.From) {
			return
		}
		// Sem numero, nome ou texto do contato no log: e dado pessoal (LGPD/GDPR) e log
		// costuma ir para lugares com menos protecao que o banco. So o tipo e a sessao.
		slog.Debug("message received", "instance", name, "type", msg.Type, "from_me", msg.FromMe)

		if fwd, on := inst.webhookOn(); on {
			fwd.Forward(msg)
		}
	})

	wp.OnPresence(func(from string, composing bool) {
		if fwd, on := inst.webhookOn(); on && !isGroup(from) {
			fwd.ReportTyping(from, composing)
		}
	})

	m.instances[name] = inst
	m.mu.Unlock()

	if persist {
		m.save(inst)
	}
	log.Printf("[manager] Instance '%s' created", name)
	return inst, nil
}

var validName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidName diz se o nome serve para instancia. O nome vira arquivo ({nome}.db) e parte de
// URL: sem essa regra, "../x" gravaria a sessao fora da pasta de dados.
func ValidName(name string) bool { return validName.MatchString(name) }

func (m *Manager) Connect(name string) (*Instance, error) {
	inst, err := m.Get(name)
	if err != nil {
		return nil, err
	}

	if err := inst.Provider.Connect(); err != nil {
		return nil, fmt.Errorf("connect '%s': %w", name, err)
	}

	return inst, nil
}

func (m *Manager) Get(name string) (*Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.instances[name]
	if !ok {
		return nil, fmt.Errorf("instance '%s' %w", name, ErrNotFound)
	}
	return inst, nil
}

// Delete apaga a instancia de verdade: desfaz o pareamento no celular (o aparelho some da
// lista), fecha a sessao e remove os arquivos. Antes so tirava da memoria, e o <nome>.db
// fazia a instancia voltar no proximo reinicio.
func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	inst, ok := m.instances[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("instance '%s' %w", name, ErrNotFound)
	}
	delete(m.instances, name)
	m.mu.Unlock()

	if err := inst.Provider.Unlink(); err != nil {
		log.Printf("[manager] unlink '%s' (continuing): %v", name, err)
	}
	_ = inst.Provider.Close()
	for _, f := range []string{m.dbFile(name), m.dbFile(name) + "-wal", m.dbFile(name) + "-shm", m.metaFile(name)} {
		if err := os.Remove(f); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("[manager] remove %s: %v", filepath.Base(f), err)
		}
	}
	log.Printf("[manager] Instance '%s' deleted", name)
	return nil
}

// Shutdown fecha todas as sessoes (gateway desligando): o SQLite de cada uma fecha limpo e
// nada fica pela metade no disco.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, inst := range m.instances {
		if err := inst.Provider.Close(); err != nil {
			log.Printf("[manager] closing '%s': %v", name, err)
		}
	}
}

func (m *Manager) Logout(name string) error {
	inst, err := m.Get(name)
	if err != nil {
		return err
	}
	return inst.Provider.Disconnect()
}

// List devolve as instancias em ordem alfabetica.
func (m *Manager) List() []InstanceInfo {
	m.mu.RLock()
	insts := make([]*Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		insts = append(insts, inst)
	}
	m.mu.RUnlock()

	list := make([]InstanceInfo, 0, len(insts))
	for _, inst := range insts {
		list = append(list, inst.info())
	}
	sort.Slice(list, func(a, b int) bool { return list[a].Name < list[b].Name })
	return list
}

// Info devolve os dados de uma instancia.
func (m *Manager) Info(name string) (InstanceInfo, error) {
	inst, err := m.Get(name)
	if err != nil {
		return InstanceInfo{}, err
	}
	return inst.info(), nil
}

// WebhookUpdate sao as mudancas pedidas no webhook; campo nil fica como esta.
type WebhookUpdate struct {
	URL     *string
	Events  []string
	Enabled *bool
}

// UpdateWebhook muda o webhook da instancia e grava. URL nova cria um forwarder novo, mas
// o historico de entregas continua o mesmo.
func (m *Manager) UpdateWebhook(name string, u WebhookUpdate) (WebhookConfig, error) {
	inst, err := m.Get(name)
	if err != nil {
		return WebhookConfig{}, err
	}
	if u.URL != nil && *u.URL != "" {
		if err := webhook.ValidateURL(*u.URL); err != nil {
			return WebhookConfig{}, err
		}
	}

	inst.mu.Lock()
	if u.URL != nil && *u.URL != inst.webhookCfg.URL {
		inst.webhookCfg.URL = *u.URL
		inst.forwarder = webhook.NewForwarder(*u.URL, m.cfg.WebhookFormat, m.cfg.WebhookSecret).
			WithHistory(inst.forwarder.History())
	}
	if len(u.Events) > 0 {
		inst.webhookCfg.Events = append([]string(nil), u.Events...)
	}
	if u.Enabled != nil {
		inst.webhookCfg.Enabled = *u.Enabled
	}
	inst.mu.Unlock()

	m.save(inst)
	return inst.WebhookConfig(), nil
}

// UpdateSettings aplica mudancas nas configuracoes e grava.
func (m *Manager) UpdateSettings(name string, apply func(*Settings) error) (Settings, error) {
	inst, err := m.Get(name)
	if err != nil {
		return Settings{}, err
	}
	inst.mu.Lock()
	s := inst.settings
	if err := apply(&s); err != nil {
		inst.mu.Unlock()
		return Settings{}, err
	}
	inst.settings = s
	inst.mu.Unlock()

	m.save(inst)
	return s, nil
}

// TestWebhook manda um evento de teste para o webhook da instancia.
func (m *Manager) TestWebhook(name string) (webhook.Delivery, error) {
	inst, err := m.Get(name)
	if err != nil {
		return webhook.Delivery{}, err
	}
	return inst.Forwarder().Test(name), nil
}

// Deliveries sao as ultimas entregas de webhook da instancia, da mais nova para a mais antiga.
func (m *Manager) Deliveries(name string) ([]webhook.Delivery, error) {
	inst, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	return inst.Forwarder().History().List(), nil
}

// ExportWebhooks devolve a configuracao de webhook de todas as instancias, por nome.
func (m *Manager) ExportWebhooks() map[string]WebhookConfig {
	out := map[string]WebhookConfig{}
	for _, info := range m.List() {
		out[info.Name] = info.Webhook
	}
	return out
}

// ImportResult diz o que a importacao fez com cada instancia.
type ImportResult struct {
	Updated []string          `json:"updated"`
	Skipped map[string]string `json:"skipped"` // nome -> motivo
}

// ImportWebhooks aplica configuracoes exportadas as instancias que existem. Instancia que
// nao existe ou URL invalida e pulada com o motivo; nada e criado.
func (m *Manager) ImportWebhooks(cfgs map[string]WebhookConfig) ImportResult {
	res := ImportResult{Updated: []string{}, Skipped: map[string]string{}}
	names := make([]string, 0, len(cfgs))
	for n := range cfgs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cfgs[name]
		url, enabled := c.URL, c.Enabled
		if _, err := m.UpdateWebhook(name, WebhookUpdate{URL: &url, Events: c.Events, Enabled: &enabled}); err != nil {
			res.Skipped[name] = err.Error()
			continue
		}
		res.Updated = append(res.Updated, name)
	}
	return res
}

func isGroup(jid string) bool {
	return len(jid) > 5 && jid[len(jid)-5:] == "@g.us"
}
