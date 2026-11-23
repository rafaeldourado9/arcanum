package provider

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

// TestToJIDPreservesServerForFullJIDs is a regression test for a production
// bug: incoming senders are increasingly identified by LID (a numeric
// namespace distinct from phone numbers, JID server "lid") rather than the
// classic phone-number JID ("s.whatsapp.net"). toJID used to strip everything
// but digits and always rebuild a "@s.whatsapp.net" JID, so replying to a LID
// sender silently targeted a nonexistent phone-number account — the message
// never arrived, with no visible error to the caller.
func TestToJIDPreservesServerForFullJIDs(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantUser   string
		wantServer string
	}{
		{
			name:       "full LID JID is parsed as-is",
			input:      "104200855892049@lid",
			wantUser:   "104200855892049",
			wantServer: "lid",
		},
		{
			name:       "full phone JID is parsed as-is",
			input:      "5511999999999@s.whatsapp.net",
			wantUser:   "5511999999999",
			wantServer: "s.whatsapp.net",
		},
		{
			name:       "bare phone number falls back to phone JID",
			input:      "5511999999999",
			wantUser:   "5511999999999",
			wantServer: "s.whatsapp.net",
		},
		{
			name:       "bare phone number with formatting is cleaned",
			input:      "+55 (11) 99999-9999",
			wantUser:   "5511999999999",
			wantServer: "s.whatsapp.net",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jid := toJID(tc.input)
			if jid.User != tc.wantUser {
				t.Errorf("User = %q, want %q", jid.User, tc.wantUser)
			}
			if jid.Server != tc.wantServer {
				t.Errorf("Server = %q, want %q", jid.Server, tc.wantServer)
			}
		})
	}
}

// Quem escreve de um aparelho vinculado chega como "123:30@lid". Responder para esse
// endereco falha no WhatsApp; a resposta tem de ir para a pessoa ("123@lid").
func TestToJIDDropsTheDevicePart(t *testing.T) {
	jid := toJID("23927447416901:30@lid")
	if jid.String() != "23927447416901@lid" || jid.Device != 0 {
		t.Fatalf("got %q (device %d), want 23927447416901@lid", jid.String(), jid.Device)
	}
}

// O sistema trabalha com telefone: remetente "@lid" vira o telefone que o WhatsApp manda junto.
func TestSenderAddressPrefersThePhoneOverTheLID(t *testing.T) {
	w := NewWhatsmeow(t.TempDir())
	lid := types.NewJID("123456789012345", types.HiddenUserServer)
	phone := types.NewJID("5567998830491", types.DefaultUserServer)
	cases := []struct {
		name string
		info types.MessageInfo
		want string
	}{
		{"telefone ja vem direto", info(phone, types.EmptyJID), "5567998830491@s.whatsapp.net"},
		{"lid com o telefone em SenderAlt", info(lid, phone), "5567998830491@s.whatsapp.net"},
		{"lid sem telefone conhecido fica lid", info(lid, types.EmptyJID), "123456789012345@lid"},
	}
	for _, tc := range cases {
		if got := w.senderAddress(tc.info.MessageSource); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func info(sender, alt types.JID) types.MessageInfo {
	var i types.MessageInfo
	i.Sender, i.SenderAlt = sender, alt
	return i
}
