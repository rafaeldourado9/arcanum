// Package instance gerencia multiplas conexoes WhatsApp simultaneas.
// Cada instancia possui seu proprio provider (whatsmeow), rate limiter,
// webhook forwarder, e configuracoes independentes.
// Thread-safe via sync.RWMutex — seguro para acesso concorrente pelos handlers HTTP.
package instance

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/rafaeldourado9/arcanum/internal/antiban"
	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/provider"
	"github.com/rafaeldourado9/arcanum/internal/webhook"
)

type Instance struct {
	Name     string                    `json:"instanceName"`
	Provider provider.WhatsAppProvider `json:"-"`
	Limiter  *antiban.RateLimiter      `json:"-"`
	Webhook  *webhook.Forwarder        `json:"-"`
	Settings *Settings                 `json:"settings"`
	WebhookConfig *WebhookConfig       `json:"webhook"`
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
	Name       string                   `json:"instanceName"`
	Status     provider.ConnectionStatus `json:"status"`
	Settings   *Settings                 `json:"settings"`
	Webhook    *WebhookConfig            `json:"webhook"`
}

type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	cfg       *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		cfg:       cfg,
	}
}

func (m *Manager) Create(name string, webhookURL string, events []string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.instances[name]; exists {
		return nil, fmt.Errorf("instance '%s' already exists", name)
	}

	dbPath := filepath.Join(m.cfg.DBPath, name+".db")
	wp := provider.NewWhatsmeow(dbPath)

	whCfg := &WebhookConfig{
		URL:     webhookURL,
		Events:  events,
		Enabled: webhookURL != "",
	}
	if whCfg.URL == "" {
		whCfg.URL = m.cfg.WebhookForwardURL
		whCfg.Enabled = m.cfg.WebhookForwardURL != ""
	}
	if len(whCfg.Events) == 0 {
		whCfg.Events = []string{"messages", "status"}
	}

	fwd := webhook.NewForwarder(whCfg.URL, m.cfg.WebhookSecret)
	rl := antiban.NewRateLimiter(m.cfg.RateLimitPerMin)

	inst := &Instance{
		Name:     name,
		Provider: wp,
		Limiter:  rl,
		Webhook:  fwd,
		Settings: &Settings{ReadMessages: true},
		WebhookConfig: whCfg,
	}

	wp.OnMessage(func(msg provider.IncomingMessage) {
		if inst.Settings.GroupsIgnore && isGroup(msg.From) {
			return
		}
		preview := msg.Text
		if preview == "" {
			preview = fmt.Sprintf("[%s]", msg.Type)
		}
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		pn := msg.PushName
		if pn == "" {
			pn = msg.From
		}
		log.Printf("[%s] %s: %s", name, pn, preview)

		if inst.WebhookConfig.Enabled {
			inst.Webhook.Forward(msg)
		}
	})

	m.instances[name] = inst
	log.Printf("[manager] Instance '%s' created", name)
	return inst, nil
}

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
		return nil, fmt.Errorf("instance '%s' not found", name)
	}
	return inst, nil
}

func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.instances[name]
	if !ok {
		return fmt.Errorf("instance '%s' not found", name)
	}

	_ = inst.Provider.Disconnect()
	delete(m.instances, name)
	log.Printf("[manager] Instance '%s' deleted", name)
	return nil
}

func (m *Manager) Logout(name string) error {
	inst, err := m.Get(name)
	if err != nil {
		return err
	}
	return inst.Provider.Disconnect()
}

func (m *Manager) List() []InstanceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]InstanceInfo, 0, len(m.instances))
	for _, inst := range m.instances {
		list = append(list, InstanceInfo{
			Name:     inst.Name,
			Status:   inst.Provider.Status(),
			Settings: inst.Settings,
			Webhook:  inst.WebhookConfig,
		})
	}
	return list
}

func isGroup(jid string) bool {
	return len(jid) > 5 && jid[len(jid)-5:] == "@g.us"
}
