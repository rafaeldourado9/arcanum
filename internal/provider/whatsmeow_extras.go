package provider

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Configuracoes por sessao que dependem do protocolo: recusar chamadas e ficar "online".

// onlineKeepAlive e o intervalo do "estou online" quando a sessao esta em modo sempre online.
const onlineKeepAlive = 60 * time.Second

var (
	_ CallSupport  = (*WhatsmeowProvider)(nil)
	_ OnlineKeeper = (*WhatsmeowProvider)(nil)
)

// OnCall registra o callback das chamadas recebidas.
func (w *WhatsmeowProvider) OnCall(handler CallHandler) {
	w.mu.Lock()
	w.callHandler = handler
	w.mu.Unlock()
}

// RejectCall recusa a chamada recebida (quem ligou ve "recusada").
func (w *WhatsmeowProvider) RejectCall(from, callID string) error {
	client := w.currentClient()
	if client == nil || w.Status() != StatusConnected {
		return fmt.Errorf("not connected")
	}
	return client.RejectCall(context.Background(), toJID(from), callID)
}

func (w *WhatsmeowProvider) handleCall(evt *events.CallOffer) {
	w.mu.RLock()
	handler := w.callHandler
	w.mu.RUnlock()
	if handler == nil || evt.CallID == "" {
		return
	}
	handler(evt.CallCreator.ToNonAD().String(), evt.CallID)
}

// SetAlwaysOnline liga ou desliga o modo sempre online: enquanto conectada, a sessao reenvia
// a presenca "disponivel" a cada minuto. Desligado, vale so o "disponivel" enviado ao
// conectar (o WhatsApp precisa dele para entregar o "digitando..." dos contatos).
func (w *WhatsmeowProvider) SetAlwaysOnline(on bool) {
	w.mu.Lock()
	w.alwaysOnline = on
	client := w.client
	connected := w.status == StatusConnected
	w.mu.Unlock()
	w.syncOnline()
	if on && connected && client != nil {
		go func() { _ = client.SendPresence(context.Background(), types.PresenceAvailable) }()
	}
}

// syncOnline reconcilia o keep-alive com o estado atual: para o anterior e, se o modo estiver
// ligado e a sessao conectada, inicia um novo.
func (w *WhatsmeowProvider) syncOnline() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.onlineCancel != nil {
		w.onlineCancel()
		w.onlineCancel = nil
	}
	if !w.alwaysOnline || w.client == nil || w.status != StatusConnected {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.onlineCancel = cancel
	client := w.client
	go func() {
		t := time.NewTicker(onlineKeepAlive)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if w.Status() == StatusConnected {
					_ = client.SendPresence(ctx, types.PresenceAvailable)
				}
			}
		}
	}()
}

// stopOnline encerra o keep-alive (ao desconectar ou fechar a sessao).
func (w *WhatsmeowProvider) stopOnline() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.onlineCancel != nil {
		w.onlineCancel()
		w.onlineCancel = nil
	}
}
