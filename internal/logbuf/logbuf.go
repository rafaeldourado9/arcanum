// Package logbuf guarda as ultimas linhas de log em memoria para o painel mostrar (lista e
// ao vivo). E um slog.Handler que repassa tudo ao handler normal (stdout) e copia para um
// anel de tamanho fixo. Atributos com nome de segredo sao mascarados na copia.
package logbuf

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Entry e uma linha de log como o painel recebe.
type Entry struct {
	Seq   uint64         `json:"seq"`
	Time  time.Time      `json:"time"`
	Level string         `json:"level"`
	Msg   string         `json:"msg"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// Buffer e o anel de linhas mais os assinantes do "ao vivo".
type Buffer struct {
	mu    sync.Mutex
	items []Entry
	next  int
	full  bool
	seq   uint64
	subs  map[chan Entry]struct{}
}

func New(size int) *Buffer {
	return &Buffer{items: make([]Entry, size), subs: map[chan Entry]struct{}{}}
}

func (b *Buffer) add(e Entry) {
	b.mu.Lock()
	b.seq++
	e.Seq = b.seq
	b.items[b.next] = e
	b.next = (b.next + 1) % len(b.items)
	if b.next == 0 {
		b.full = true
	}
	for ch := range b.subs {
		select {
		case ch <- e:
		default: // assinante lento perde a linha; o log nunca trava por causa do painel
		}
	}
	b.mu.Unlock()
}

// Recent devolve ate limit linhas, da mais antiga para a mais nova, com nivel >= min.
func (b *Buffer) Recent(limit int, min slog.Level) []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := b.next
	if b.full {
		n = len(b.items)
	}
	out := make([]Entry, 0, n)
	for i := 0; i < n; i++ {
		idx := i
		if b.full {
			idx = (b.next + i) % len(b.items)
		}
		e := b.items[idx]
		if levelOf(e.Level) >= min {
			out = append(out, e)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// Subscribe recebe cada linha nova ate cancel ser chamado.
func (b *Buffer) Subscribe() (<-chan Entry, func()) {
	ch := make(chan Entry, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
	}
}

// ParseLevel aceita "debug", "info", "warn", "error" (padrao: debug, tudo).
func ParseLevel(s string) slog.Level { return levelOf(strings.ToUpper(s)) }

func levelOf(s string) slog.Level {
	switch s {
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

// Handler repassa ao inner e copia para o Buffer.
type Handler struct {
	inner  slog.Handler
	buf    *Buffer
	attrs  []slog.Attr
	prefix string
}

func NewHandler(inner slog.Handler, buf *Buffer) *Handler {
	return &Handler{inner: inner, buf: buf}
}

func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool { return h.inner.Enabled(ctx, l) }

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	err := h.inner.Handle(ctx, r)
	e := Entry{Time: r.Time, Level: r.Level.String(), Msg: r.Message}
	if len(h.attrs) > 0 || r.NumAttrs() > 0 {
		e.Attrs = map[string]any{}
		for _, a := range h.attrs {
			put(e.Attrs, h.prefix, a)
		}
		r.Attrs(func(a slog.Attr) bool {
			put(e.Attrs, h.prefix, a)
			return true
		})
	}
	h.buf.add(e)
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{inner: h.inner.WithAttrs(attrs), buf: h.buf, attrs: append(append([]slog.Attr{}, h.attrs...), attrs...), prefix: h.prefix}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{inner: h.inner.WithGroup(name), buf: h.buf, attrs: h.attrs, prefix: h.prefix + name + "."}
}

// secretWords: atributo cujo nome contem uma destas palavras aparece mascarado no painel
// (a chave gerada no boot, por exemplo, sai no stdout para o operador, mas nao no painel).
var secretWords = []string{"key", "secret", "password", "token", "authorization", "cookie"}

func put(m map[string]any, prefix string, a slog.Attr) {
	k := prefix + a.Key
	lower := strings.ToLower(k)
	for _, w := range secretWords {
		if strings.Contains(lower, w) {
			m[k] = "[redacted]"
			return
		}
	}
	m[k] = a.Value.Resolve().Any()
}
