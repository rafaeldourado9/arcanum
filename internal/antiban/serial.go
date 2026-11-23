package antiban

import "sync"

// ChatQueue roda os envios de um mesmo chat um depois do outro, na ordem de chegada.
//
// O envio humanizado dorme um tempo sorteado antes de mandar. Com uma goroutine solta por
// mensagem, a resposta que o backend quebra em partes ("Oi, Ana!", "Sobre o orçamento...")
// podia chegar fora de ordem. Chats diferentes continuam em paralelo.
type ChatQueue struct {
	mu    sync.Mutex
	tails map[string]chan struct{}
}

func NewChatQueue() *ChatQueue {
	return &ChatQueue{tails: map[string]chan struct{}{}}
}

// Run agenda job depois do último envio pendente do mesmo chat e volta na hora.
// A ordem vale pela ordem das chamadas a Run: o backend só manda a parte seguinte depois
// da resposta 202 da anterior.
func (q *ChatQueue) Run(chat string, job func()) {
	q.mu.Lock()
	previous := q.tails[chat]
	done := make(chan struct{})
	q.tails[chat] = done
	q.mu.Unlock()

	go func() {
		defer q.finish(chat, done)
		if previous != nil {
			<-previous
		}
		job()
	}()
}

func (q *ChatQueue) finish(chat string, done chan struct{}) {
	close(done)
	q.mu.Lock()
	if q.tails[chat] == done {
		delete(q.tails, chat) // ninguém esperando: o mapa não cresce com chats antigos
	}
	q.mu.Unlock()
}
