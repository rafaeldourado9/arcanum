package webhook

import (
	"sync"
	"time"
)

// Delivery e uma tentativa de entrega de webhook, para o painel mostrar o que saiu e o que
// falhou. Nao guarda o corpo (tem dado pessoal), so o evento e o resultado.
type Delivery struct {
	Time       time.Time `json:"time"`
	Event      string    `json:"event"`
	Status     int       `json:"status"`          // codigo HTTP; 0 quando nem conectou
	OK         bool      `json:"ok"`              // 2xx
	Error      string    `json:"error,omitempty"` // erro de rede, quando houve
	DurationMs int64     `json:"durationMs"`
}

// History guarda as ultimas entregas de uma instancia (anel de tamanho fixo).
type History struct {
	mu    sync.Mutex
	items []Delivery
	next  int
	full  bool
}

const historySize = 100

func NewHistory() *History { return &History{items: make([]Delivery, historySize)} }

func (h *History) add(d Delivery) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.items[h.next] = d
	h.next = (h.next + 1) % len(h.items)
	if h.next == 0 {
		h.full = true
	}
}

// List devolve as entregas da mais nova para a mais antiga.
func (h *History) List() []Delivery {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := h.next
	if h.full {
		n = len(h.items)
	}
	out := make([]Delivery, 0, n)
	for i := 0; i < n; i++ {
		idx := (h.next - 1 - i + len(h.items)) % len(h.items)
		out = append(out, h.items[idx])
	}
	return out
}
