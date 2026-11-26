// Package audit registra as acoes sensiveis (login, criar/apagar sessao, mudar webhook, criar
// ou revogar chave, acesso negado) em um arquivo so de acrescimo, uma linha JSON por evento.
//
// Nunca entram aqui: chaves, senhas, segredos de webhook, texto de mensagens nem telefones de
// contatos. So quem fez (chave ou painel), de onde, o que e em qual sessao.
package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// maxBytes: ao passar disso o arquivo atual vira <arquivo>.1 (um unico arquivo antigo e
// mantido), para o log nunca encher o disco.
const maxBytes = 5 << 20

// Event e uma linha do log.
type Event struct {
	Time   time.Time `json:"time"`
	Actor  string    `json:"actor"`            // "admin", "panel", "key:<id>" ou "-" (nao autenticado)
	IP     string    `json:"ip"`               // IP da conexao (nao confia em X-Forwarded-For)
	Action string    `json:"action"`           // ex.: "instance.create", "auth.denied"
	Target string    `json:"target,omitempty"` // sessao ou chave afetada
	Status int       `json:"status,omitempty"` // codigo HTTP da resposta
	Detail string    `json:"detail,omitempty"` // texto curto, sem dado pessoal
}

// Log e o arquivo de auditoria. O valor nil e valido e nao grava nada.
type Log struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
	now  func() time.Time
}

// Open abre (ou cria, 0600) o arquivo de auditoria.
func Open(path string) (*Log, error) {
	l := &Log{path: path, now: time.Now}
	if err := l.openLocked(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Log) openLocked() error {
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- caminho vem da configuracao do operador
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	l.f, l.size = f, st.Size()
	return nil
}

// Record grava um evento. Falha de disco nao derruba a requisicao: so deixa de registrar.
func (l *Log) Record(e Event) {
	if l == nil {
		return
	}
	if e.Time.IsZero() {
		e.Time = l.now().UTC()
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	b = append(b, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return
	}
	if l.size+int64(len(b)) > maxBytes {
		_ = l.f.Close()
		_ = os.Rename(l.path, l.path+".1")
		if err := l.openLocked(); err != nil {
			l.f = nil
			return
		}
	}
	n, _ := l.f.Write(b)
	l.size += int64(n)
}

// Tail devolve os ultimos n eventos, do mais novo para o mais antigo.
func (l *Log) Tail(n int) []Event {
	out := []Event{}
	if l == nil || n <= 0 {
		return out
	}
	l.mu.Lock()
	path := l.path
	l.mu.Unlock()

	f, err := os.Open(path) // #nosec G304 -- caminho vem da configuracao do operador
	if err != nil {
		return out
	}
	defer func() { _ = f.Close() }()

	var all []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			all = append(all, e)
		}
	}
	for i := len(all) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, all[i])
	}
	return out
}

// Close fecha o arquivo.
func (l *Log) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		_ = l.f.Close()
		l.f = nil
	}
}
