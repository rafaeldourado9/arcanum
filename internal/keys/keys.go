// Package keys guarda as chaves de API com escopo: cada integrador recebe uma chave que
// enxerga so as sessoes dele (e, se quiser, so le), sem a chave mestra do gateway.
//
// Decisoes de seguranca:
//   - A chave inteira aparece uma unica vez, na criacao. Em disco fica so o SHA-256 dela
//     (chave de 256 bits aleatorios: hash simples basta, nao precisa de KDF lento).
//   - Revogar e imediato e a chave revogada continua listada (rastro para a auditoria).
//   - Arquivo 0600, escrita atomica.
package keys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// Prefix identifica uma chave do Arcanum (ajuda scanners de segredo e quem le um log).
	Prefix  = "arc_"
	maxKeys = 1000
)

var (
	// ErrNotFound: nao existe chave com esse id.
	ErrNotFound = errors.New("key not found")

	validName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

// Key e a parte publica de uma chave (nunca contem o segredo nem o hash).
type Key struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Hint      string    `json:"hint"`      // inicio da chave, so para reconhece-la numa lista
	Instances []string  `json:"instances"` // sessoes que a chave enxerga
	ReadOnly  bool      `json:"readOnly"`  // so consulta: nao envia, nao muda configuracao
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  time.Time `json:"lastUsed,omitempty"`
	Revoked   bool      `json:"revoked"`
}

// Allows diz se a chave enxerga a sessao.
func (k Key) Allows(instance string) bool {
	for _, n := range k.Instances {
		if n == instance {
			return true
		}
	}
	return false
}

type record struct {
	Key
	Hash string `json:"hash"`
}

// Store e o cadastro de chaves, gravado em um arquivo JSON.
type Store struct {
	mu     sync.Mutex
	path   string
	recs   []*record
	byHash map[string]*record
	now    func() time.Time
}

// Open carrega o cadastro (arquivo ausente = vazio) e cria o arquivo na primeira gravacao.
func Open(path string) (*Store, error) {
	s := &Store{path: path, byHash: map[string]*record{}, now: time.Now}
	b, err := os.ReadFile(path) // #nosec G304 -- caminho vem da configuracao do operador
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.recs); err != nil {
		return nil, fmt.Errorf("corrupt key file %s: %w", path, err)
	}
	for _, r := range s.recs {
		s.byHash[r.Hash] = r
	}
	return s, nil
}

func hashOf(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create cadastra uma chave e devolve o segredo, que nao se recupera depois.
func (s *Store) Create(name string, instances []string, readOnly bool) (string, Key, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return "", Key{}, errors.New("name is required (up to 64 characters)")
	}
	if len(instances) == 0 {
		return "", Key{}, errors.New("instances is required: list the sessions this key may use")
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(instances))
	for _, n := range instances {
		if !validName.MatchString(n) {
			return "", Key{}, fmt.Errorf("invalid instance name %q", n)
		}
		if !seen[n] {
			seen[n] = true
			clean = append(clean, n)
		}
	}
	sort.Strings(clean)

	body, err := randomHex(32)
	if err != nil {
		return "", Key{}, err
	}
	id, err := randomHex(6)
	if err != nil {
		return "", Key{}, err
	}
	secret := Prefix + body
	rec := &record{
		Key: Key{
			ID:        "key_" + id,
			Name:      name,
			Hint:      secret[:len(Prefix)+4],
			Instances: clean,
			ReadOnly:  readOnly,
			CreatedAt: s.now().UTC(),
		},
		Hash: hashOf(secret),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.recs) >= maxKeys {
		return "", Key{}, errors.New("too many keys: revoke the ones you no longer use")
	}
	s.recs = append(s.recs, rec)
	s.byHash[rec.Hash] = rec
	if err := s.saveLocked(); err != nil {
		s.recs = s.recs[:len(s.recs)-1]
		delete(s.byHash, rec.Hash)
		return "", Key{}, err
	}
	return secret, rec.Key, nil
}

// Authenticate devolve a chave dona do segredo, se existir e nao estiver revogada.
func (s *Store) Authenticate(secret string) (Key, bool) {
	if s == nil || !strings.HasPrefix(secret, Prefix) {
		return Key{}, false
	}
	h := hashOf(secret)
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.byHash[h]
	if !ok || r.Revoked {
		return Key{}, false
	}
	r.LastUsed = s.now().UTC() // so em memoria: gravar em disco a cada pedido seria caro
	return r.Key, true
}

// List devolve todas as chaves (inclusive as revogadas), da mais antiga para a mais nova.
func (s *Store) List() []Key {
	if s == nil {
		return []Key{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Key, 0, len(s.recs))
	for _, r := range s.recs {
		out = append(out, r.Key)
	}
	return out
}

// Revoke desativa a chave na hora.
func (s *Store) Revoke(id string) (Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.recs {
		if r.ID == id {
			r.Revoked = true
			if err := s.saveLocked(); err != nil {
				r.Revoked = false
				return Key{}, err
			}
			return r.Key, nil
		}
	}
	return Key{}, ErrNotFound
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.recs, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
