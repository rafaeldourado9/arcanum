package antiban

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestChatQueueKeepsOrderWithinAChat(t *testing.T) {
	q := NewChatQueue()
	var mu sync.Mutex
	var got []int
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		n := i
		// Primeiro envio dorme mais que os outros: sem fila, chegaria por último.
		wait := time.Duration(rand.Intn(5)+(5-i)*10) * time.Millisecond
		q.Run("5567999990000", func() {
			defer wg.Done()
			time.Sleep(wait)
			mu.Lock()
			got = append(got, n)
			mu.Unlock()
		})
	}
	wg.Wait()
	for i, n := range got {
		if n != i {
			t.Fatalf("fora de ordem: %v", got)
		}
	}
}

func TestChatQueueRunsDifferentChatsInParallel(t *testing.T) {
	q := NewChatQueue()
	started := make(chan string, 2)
	release := make(chan struct{})
	q.Run("a", func() { started <- "a"; <-release })
	q.Run("b", func() { started <- "b"; <-release })
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("um chat esperou o outro")
		}
	}
	close(release)
}

func TestChatQueueForgetsIdleChats(t *testing.T) {
	q := NewChatQueue()
	var wg sync.WaitGroup
	wg.Add(1)
	q.Run("a", wg.Done)
	wg.Wait()
	time.Sleep(10 * time.Millisecond)
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.tails) != 0 {
		t.Fatalf("chat parado ficou no mapa: %d", len(q.tails))
	}
}
