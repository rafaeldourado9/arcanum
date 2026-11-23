package provider

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// So conversa de uma pessoa com a empresa chega ao backend. Status (stories), grupos,
// listas de transmissao, canais e eventos sem conteudo (reacao, mensagem apagada) nao:
// antes o bot respondia o menu a quem so tinha postado um status.
func TestHandleMessageForwardsOnlyPeopleTyping(t *testing.T) {
	person := types.NewJID("5567999990000", types.DefaultUserServer)
	text := &waE2E.Message{Conversation: proto.String("oi")}
	reaction := &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{Text: proto.String("👍")}}

	now := time.Now()
	cases := []struct {
		name    string
		chat    types.JID
		message *waE2E.Message
		sent    time.Time
		want    bool
	}{
		{"texto de uma pessoa", person, text, now, true},
		{"pendencia antiga entregue ao parear", person, text, now.Add(-time.Hour), false},
		{"status (stories)", types.StatusBroadcastJID, text, now, false},
		{"grupo", types.NewJID("123456789", types.GroupServer), text, now, false},
		{"canal", types.NewJID("120363", types.NewsletterServer), text, now, false},
		{"lista de transmissao", types.NewJID("1695", types.BroadcastServer), text, now, false},
		{"reacao", person, reaction, now, false},
	}
	// Remetente de aparelho vinculado: o backend recebe a pessoa, sem o ":30".
	t.Run("aparelho vinculado vira a pessoa", func(t *testing.T) {
		var from string
		w := &WhatsmeowProvider{handler: func(m IncomingMessage) { from = m.From }}
		device := types.JID{User: "23927447416901", Server: types.HiddenUserServer, Device: 30}
		w.handleMessage(&events.Message{
			Info:    types.MessageInfo{MessageSource: types.MessageSource{Chat: device.ToNonAD(), Sender: device}, ID: "m2", Timestamp: now},
			Message: text,
		})
		if from != "23927447416901@lid" {
			t.Fatalf("from=%q, want 23927447416901@lid", from)
		}
	})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := false
			w := &WhatsmeowProvider{handler: func(IncomingMessage) { got = true }}
			w.handleMessage(&events.Message{
				Info:    types.MessageInfo{MessageSource: types.MessageSource{Chat: c.chat, Sender: person}, ID: "m1", Timestamp: c.sent},
				Message: c.message,
			})
			if got != c.want {
				t.Fatalf("repassou=%v, esperado %v", got, c.want)
			}
		})
	}
}
