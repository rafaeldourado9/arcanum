package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/media"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite" // SQLite em Go puro: binario estatico, sem CGO, compila para qualquer SO
)

// WhatsApp's Linked Devices screen picks an icon from a fixed set tied to
// PlatformType (browser logos, phone, desktop, ...) — there's no way to show
// a custom logo image. DESKTOP gives a generic computer icon next to our name
// instead of the "Outro dispositivo"/question-mark fallback for UNKNOWN.
func init() {
	store.DeviceProps.Os = proto.String("Arcanum")
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_DESKTOP.Enum()
}

type WhatsmeowProvider struct {
	dbPath            string
	client            *whatsmeow.Client
	container         *sqlstore.Container
	status            ConnectionStatus
	qrCode            string
	handler           MessageHandler
	presence          PresenceHandler
	subscribed        map[types.JID]bool // contatos cujo "digitando..." ja foi pedido nesta conexao
	mu                sync.RWMutex
	reconnectAttempts int
	reconnecting      bool
}

func NewWhatsmeow(dbPath string) *WhatsmeowProvider {
	return &WhatsmeowProvider{
		dbPath: dbPath,
		status: StatusDisconnected,
	}
}

func (w *WhatsmeowProvider) Name() string { return "whatsmeow" }

func (w *WhatsmeowProvider) Status() ConnectionStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// PhoneNumber is the paired phone (digits only), empty before pairing. The backend uses it to
// refuse a number that is already connected to another company.
func (w *WhatsmeowProvider) PhoneNumber() string {
	w.mu.RLock()
	client := w.client
	w.mu.RUnlock()
	if client == nil || client.Store == nil || client.Store.ID == nil {
		return ""
	}
	return client.Store.ID.User
}

func (w *WhatsmeowProvider) QRCode() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.qrCode
}

func (w *WhatsmeowProvider) setStatus(s ConnectionStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status = s
}

func (w *WhatsmeowProvider) setQR(qr string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.qrCode = qr
}

// Connect abre (ou reabre) a sessao. Chamado de novo enquanto ja conecta ou espera a
// leitura do QR, nao faz nada: antes cada clique em "conectar" abria outro banco e outro
// cliente sem fechar os anteriores, e eles disputavam a mesma sessao.
func (w *WhatsmeowProvider) Connect() error {
	w.mu.Lock()
	old, oldContainer := w.client, w.container
	w.client, w.container = nil, nil
	w.mu.Unlock()

	if old != nil && old.IsConnected() {
		w.mu.Lock()
		w.client, w.container = old, oldContainer
		w.mu.Unlock()
		return nil
	}
	if old != nil {
		old.Disconnect()
	}
	if oldContainer != nil {
		_ = oldContainer.Close()
	}

	container, err := sqlstore.New(context.Background(), "sqlite", sqliteDSN(w.dbPath), waLog.Noop)
	if err != nil {
		return fmt.Errorf("sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		_ = container.Close()
		return fmt.Errorf("device store: %w", err)
	}

	// Avisos e erros do cliente no log: com Noop, recusa do WhatsApp (ex.: versão
	// desatualizada) deixava a tela sem QR e sem nenhuma pista.
	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("whatsmeow", "WARN", false))
	client.AddEventHandler(w.eventHandler)

	w.mu.Lock()
	w.client, w.container = client, container
	w.mu.Unlock()

	if client.Store.ID == nil {
		return w.connectWithQR(client)
	}

	w.setStatus(StatusConnecting)
	return client.Connect()
}

func (w *WhatsmeowProvider) connectWithQR(client *whatsmeow.Client) error {
	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return fmt.Errorf("qr channel: %w", err)
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	return w.watchQR(qrChan)
}

func (w *WhatsmeowProvider) watchQR(qrChan <-chan whatsmeow.QRChannelItem) error {
	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				w.setQR(evt.Code)
				w.setStatus(StatusQR)
				log.Println("[whatsmeow] QR code ready — scan with WhatsApp (GET /api/instance/qr/{instance}?format=png)")
			case "success":
				w.setQR("")
				log.Println("[whatsmeow] Pairing successful")
			case "timeout":
				// Ninguem leu nenhum dos QR codes: para e espera um novo "conectar". Antes
				// reconectava para sempre, gerando QR a cada 20 s numa sessao que ninguem ia
				// parear (e todo boot religava esse laco).
				log.Println("[whatsmeow] QR expired without being scanned — waiting for a new connect")
				w.setQR("")
				if c := w.currentClient(); c != nil {
					c.Disconnect()
				}
				w.setStatus(StatusDisconnected)
			default:
				// err-client-outdated, error, err-unexpected-state...: antes sumiam sem rastro
				// e a tela ficava sem QR. Registra para dar para diagnosticar.
				log.Printf("[whatsmeow] QR event %q: %v", evt.Event, evt.Error)
				w.setQR("")
				w.setStatus(StatusDisconnected)
			}
		}
	}()

	return nil
}

func (w *WhatsmeowProvider) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		w.setStatus(StatusConnected)
		w.setQR("")
		w.mu.Lock()
		w.reconnectAttempts = 0
		w.reconnecting = false
		w.subscribed = map[types.JID]bool{} // a inscricao vale so para a conexao
		w.mu.Unlock()
		log.Println("[whatsmeow] Connected")
		// O WhatsApp so entrega o "digitando..." dos contatos a quem esta online.
		go func() { _ = w.client.SendPresence(context.Background(), types.PresenceAvailable) }()

	case *events.Disconnected:
		log.Println("[whatsmeow] Disconnected")
		w.reconnect()

	case *events.LoggedOut:
		w.setStatus(StatusDisconnected)
		w.setQR("")
		log.Println("[whatsmeow] Logged out from the phone — delete the session and create it again to re-pair")

	case *events.Message:
		w.handleMessage(v)

	case *events.ChatPresence:
		w.handleChatPresence(v)

	case *events.HistorySync:
		// ignore
	}
}

// Idade maxima de uma mensagem recebida para ir ao atendimento (ver handleMessage).
const maxMessageAge = 10 * time.Minute

// senderAddress e o telefone de quem escreveu. O WhatsApp manda cada vez mais o remetente
// como id interno ("123...@lid"), num espaco numerico diferente do telefone; o sistema trabalha
// com telefone (lista "a IA nao responde", LGPD, funil, agenda). O telefone vem em SenderAlt ou
// na tabela LID -> telefone que a sessao guarda. So se nenhum dos dois souber, fica o "@lid"
// (responder para ele ainda funciona, e o JID completo e preservado em toJID).
// ToNonAD tira o aparelho ("123:30@lid" -> "123@lid"): responder para o aparelho falha.
func (w *WhatsmeowProvider) senderAddress(info types.MessageSource) string {
	sender := info.Sender.ToNonAD()
	if sender.Server != types.HiddenUserServer {
		return sender.String()
	}
	if alt := info.SenderAlt.ToNonAD(); alt.Server == types.DefaultUserServer {
		return alt.String()
	}
	if pn := w.phoneForLID(sender); !pn.IsEmpty() {
		return pn.ToNonAD().String()
	}
	return sender.String()
}

// conversationAddress e o contato da conversa, igual nos dois sentidos. Mensagem que o
// proprio numero mandou (atendente no celular) vem com Sender = nos; a chave da conversa
// tem que ser o contato (Chat), senao a aplicacao abre uma conversa "consigo mesma".
func (w *WhatsmeowProvider) conversationAddress(info types.MessageSource) string {
	if !info.IsFromMe {
		return w.senderAddress(info)
	}
	chat := info.Chat.ToNonAD()
	if chat.Server != types.HiddenUserServer {
		return chat.String()
	}
	if pn := w.phoneForLID(chat); !pn.IsEmpty() {
		return pn.ToNonAD().String()
	}
	return chat.String()
}

func (w *WhatsmeowProvider) phoneForLID(lid types.JID) types.JID {
	w.mu.RLock()
	client := w.client
	w.mu.RUnlock()
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		return types.EmptyJID
	}
	pn, err := client.Store.LIDs.GetPNForLID(context.Background(), lid)
	if err != nil {
		return types.EmptyJID
	}
	return pn
}

func (w *WhatsmeowProvider) handleMessage(evt *events.Message) {
	w.mu.RLock()
	handler := w.handler
	w.mu.RUnlock()

	if handler == nil {
		return
	}

	// So conversa de pessoa com a empresa: grupos, status (stories, em status@broadcast),
	// listas de transmissao e canais nao sao alguem escrevendo, e o bot responderia o menu.
	switch evt.Info.Chat.Server {
	case "g.us", "broadcast", "newsletter":
		return
	}
	// Pendencias entregues de uma vez ao parear (ou ao voltar de muito tempo offline): o
	// bot responderia dezenas de conversas velhas num minuto, estouraria o limite anti-
	// bloqueio e a resposta de quem esta escrevendo agora nao sairia.
	if time.Since(evt.Info.Timestamp) > maxMessageAge {
		return
	}

	msg := IncomingMessage{
		MessageID: evt.Info.ID,
		// Sempre o telefone ("5567...@s.whatsapp.net"), nunca o id interno "@lid": ver
		// senderAddress.
		From:      w.conversationAddress(evt.Info.MessageSource),
		FromMe:    evt.Info.IsFromMe,
		Timestamp: evt.Info.Timestamp.Unix(),
		PushName:  evt.Info.PushName,
		Type:      "unknown",
	}

	raw := evt.Message

	if raw.GetConversation() != "" {
		msg.Type = "text"
		msg.Text = raw.GetConversation()
	} else if raw.GetExtendedTextMessage() != nil {
		msg.Type = "text"
		msg.Text = raw.GetExtendedTextMessage().GetText()
	} else if im := raw.GetImageMessage(); im != nil {
		msg.Type = "image"
		msg.Text = im.GetCaption()
		msg.Media = w.downloadSubMedia(im, im.GetMimetype())
	} else if am := raw.GetAudioMessage(); am != nil {
		msg.Type = "audio"
		msg.Media = w.downloadSubMedia(am, am.GetMimetype())
	} else if dm := raw.GetDocumentMessage(); dm != nil {
		msg.Type = "document"
		msg.Text = dm.GetCaption()
		md := w.downloadSubMedia(dm, dm.GetMimetype())
		if md != nil {
			md.Filename = dm.GetFileName()
		}
		msg.Media = md
	} else if vm := raw.GetVideoMessage(); vm != nil {
		msg.Type = "video"
		msg.Text = vm.GetCaption()
		msg.Media = w.downloadSubMedia(vm, vm.GetMimetype())
	} else if lm := raw.GetLocationMessage(); lm != nil {
		msg.Type = "location"
		msg.Location = &Location{
			Latitude:  lm.GetDegreesLatitude(),
			Longitude: lm.GetDegreesLongitude(),
			Name:      lm.GetName(),
			Address:   lm.GetAddress(),
		}
	} else if raw.GetStickerMessage() != nil {
		msg.Type = "sticker"
	}

	// Reacao, mensagem apagada ou editada, voto em enquete, troca de chaves: sem conteudo
	// para o atendimento. Repassar faria o bot responder a um evento que ninguem digitou.
	if msg.Type == "unknown" {
		return
	}

	w.subscribeTyping(evt.Info.Sender.ToNonAD())
	handler(msg)
}

// subscribeTyping pede ao WhatsApp o "digitando..." de quem escreveu, uma vez por conexao: o
// atendimento espera o cliente terminar antes de responder (backend, evento "typing").
func (w *WhatsmeowProvider) subscribeTyping(jid types.JID) {
	w.mu.Lock()
	if w.subscribed == nil {
		w.subscribed = map[types.JID]bool{}
	}
	done := w.subscribed[jid]
	w.subscribed[jid] = true
	client := w.client
	w.mu.Unlock()
	if done || client == nil {
		return
	}
	go func() {
		if err := client.SubscribePresence(context.Background(), jid); err != nil {
			log.Printf("[whatsmeow] presence subscribe failed: %v", err)
		}
	}()
}

func (w *WhatsmeowProvider) handleChatPresence(evt *events.ChatPresence) {
	w.mu.RLock()
	handler := w.presence
	w.mu.RUnlock()
	if handler == nil || evt.IsFromMe || evt.IsGroup {
		return
	}
	handler(w.senderAddress(evt.MessageSource), evt.State == types.ChatPresenceComposing)
}

func (w *WhatsmeowProvider) downloadSubMedia(msg whatsmeow.DownloadableMessage, mimetype string) *MediaData {
	data, err := w.client.Download(context.Background(), msg)
	if err != nil {
		log.Printf("[whatsmeow] media download failed: %v", err)
		return nil
	}
	return &MediaData{
		MimeType: mimetype,
		Data:     data,
	}
}

func (w *WhatsmeowProvider) reconnect() {
	// Dedup: garante um unico loop de reconexao ativo por vez (eventos
	// Disconnected podem disparar em rajada durante uma queda de rede).
	w.mu.Lock()
	if w.reconnecting {
		w.mu.Unlock()
		return
	}
	w.reconnecting = true
	w.reconnectAttempts++
	attempts := w.reconnectAttempts
	w.mu.Unlock()

	// Backoff linear com teto de 30s. NUNCA desiste: um gateway municipal
	// precisa se auto-recuperar de falhas transitorias de rede/DNS sem
	// intervencao manual. Em LoggedOut o loop nao e acionado (sessao invalida).
	delay := time.Duration(5*attempts) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	w.setStatus(StatusConnecting)
	log.Printf("[whatsmeow] Reconnecting in %v (attempt %d)", delay, attempts)

	time.AfterFunc(delay, func() {
		if c := w.currentClient(); c != nil {
			c.Disconnect()
		}
		err := w.Connect()

		w.mu.Lock()
		w.reconnecting = false
		w.mu.Unlock()

		if err != nil {
			log.Printf("[whatsmeow] Reconnect failed: %v — retrying", err)
			w.reconnect()
		}
	})
}

func (w *WhatsmeowProvider) Disconnect() error {
	if c := w.currentClient(); c != nil {
		c.Disconnect()
	}
	return nil
}

// Close encerra a sessao de vez (desligar o gateway ou apagar a instancia): desconecta e
// fecha o banco SQLite da sessao.
func (w *WhatsmeowProvider) Close() error {
	w.mu.Lock()
	client, container := w.client, w.container
	w.client, w.container = nil, nil
	w.mu.Unlock()
	if client != nil {
		client.Disconnect()
	}
	if container != nil {
		return container.Close()
	}
	return nil
}

// Unlink tira o aparelho da lista de aparelhos conectados do celular (logout no WhatsApp).
func (w *WhatsmeowProvider) Unlink() error {
	c := w.currentClient()
	if c == nil || c.Store == nil || c.Store.ID == nil || !c.IsConnected() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return c.Logout(ctx)
}

func (w *WhatsmeowProvider) currentClient() *whatsmeow.Client {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.client
}

// toJID builds the JID to send to. If the input is already a full JID (e.g.
// "104200855892049@lid", as received from incoming messages — see OnMessage),
// it's parsed as-is so replies go back through the same identifier namespace
// the message came from. Plain digit strings (e.g. "5511999999999", as passed
// by direct API callers) are treated as phone numbers, same as before.
func toJID(to string) types.JID {
	if strings.Contains(to, "@") {
		if jid, err := types.ParseJID(to); err == nil {
			// Endereco gravado com aparelho (conversas antigas): envia para a pessoa.
			return jid.ToNonAD()
		}
	}

	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, to)
	return types.NewJID(clean, types.DefaultUserServer)
}

func (w *WhatsmeowProvider) SendText(opts SendTextOptions) (SendResult, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return SendResult{}, fmt.Errorf("not connected")
	}

	jid := toJID(opts.To)

	// For plain phone numbers, verify the recipient is on WhatsApp before
	// sending. Unregistered/invalid numbers (e.g. seed/test data) otherwise
	// fail with a permanent server error (463) only AFTER the send attempt.
	// Inbound replies arrive as @lid JIDs (already-known contacts) — skip the
	// check for those. If the check itself errors, fall through and try anyway.
	if jid.Server == types.DefaultUserServer {
		if resp, err := w.client.IsOnWhatsApp(context.Background(), []string{"+" + jid.User}); err == nil {
			registered := false
			for _, r := range resp {
				if r.IsIn {
					registered = true
					jid = r.JID
				}
			}
			if !registered {
				return SendResult{}, fmt.Errorf("recipient %s is not on WhatsApp", opts.To)
			}
		}
	}

	resp, err := w.client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(opts.Text),
	})
	if err != nil {
		return SendResult{}, err
	}

	return SendResult{
		MessageID: resp.ID,
		Provider:  "whatsmeow",
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (w *WhatsmeowProvider) SendMedia(opts SendMediaOptions) (SendResult, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return SendResult{}, fmt.Errorf("not connected")
	}

	data := opts.Data
	if len(data) == 0 && opts.URL != "" {
		return SendResult{}, fmt.Errorf("URL-based media send not supported, provide data bytes")
	}

	var mediaType whatsmeow.MediaType
	switch opts.Type {
	case "image":
		mediaType = whatsmeow.MediaImage
	case "audio":
		mediaType = whatsmeow.MediaAudio
	case "document":
		mediaType = whatsmeow.MediaDocument
	case "video":
		mediaType = whatsmeow.MediaVideo
	default:
		mediaType = whatsmeow.MediaDocument
	}

	uploaded, err := w.client.Upload(context.Background(), data, mediaType)
	if err != nil {
		return SendResult{}, fmt.Errorf("upload: %w", err)
	}

	var msg *waE2E.Message
	switch opts.Type {
	case "image":
		msg = &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(opts.Caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				Mimetype:      proto.String(opts.MimeType),
			},
		}
	case "audio":
		ptt := strings.Contains(opts.MimeType, "ogg")
		msg = &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				Mimetype:      proto.String(opts.MimeType),
				PTT:           proto.Bool(ptt),
			},
		}
	case "video":
		msg = &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(opts.Caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				Mimetype:      proto.String(opts.MimeType),
			},
		}
	default:
		msg = &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption:       proto.String(opts.Caption),
				FileName:      proto.String(opts.Filename),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				Mimetype:      proto.String(opts.MimeType),
			},
		}
	}

	resp, err := w.client.SendMessage(context.Background(), toJID(opts.To), msg)
	if err != nil {
		return SendResult{}, err
	}

	return SendResult{
		MessageID: resp.ID,
		Provider:  "whatsmeow",
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (w *WhatsmeowProvider) SendPresence(to string, presenceType string) error {
	if w.client == nil || w.Status() != StatusConnected {
		return nil
	}

	jid := toJID(to)
	ctx := context.Background()
	switch presenceType {
	case "composing":
		return w.client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	case "recording":
		return w.client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaAudio)
	case "paused":
		return w.client.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")
	case "available":
		return w.client.SendPresence(ctx, types.PresenceAvailable)
	}
	return nil
}

func (w *WhatsmeowProvider) MarkAsRead(messageID string, from string) error {
	if w.client == nil || w.Status() != StatusConnected || messageID == "" {
		return nil
	}
	return w.client.MarkRead(context.Background(), []types.MessageID{messageID}, time.Now(), toJID(from), types.EmptyJID)
}

func (w *WhatsmeowProvider) OnMessage(handler MessageHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handler = handler
}

func (w *WhatsmeowProvider) OnPresence(handler PresenceHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.presence = handler
}

func (w *WhatsmeowProvider) SendLocation(opts SendLocationOptions) (SendResult, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return SendResult{}, fmt.Errorf("not connected")
	}

	resp, err := w.client.SendMessage(context.Background(), toJID(opts.To), &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  proto.Float64(opts.Latitude),
			DegreesLongitude: proto.Float64(opts.Longitude),
			Name:             proto.String(opts.Name),
			Address:          proto.String(opts.Address),
		},
	})
	if err != nil {
		return SendResult{}, err
	}
	return SendResult{MessageID: resp.ID, Provider: "whatsmeow", Timestamp: time.Now().UnixMilli()}, nil
}

func (w *WhatsmeowProvider) SendContact(opts SendContactOptions) (SendResult, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return SendResult{}, fmt.Errorf("not connected")
	}

	var vcards []string
	for _, c := range opts.Contacts {
		vcard := fmt.Sprintf("BEGIN:VCARD\nVERSION:3.0\nFN:%s\nTEL;type=CELL:+%s\nEND:VCARD", c.FullName, c.PhoneNumber)
		vcards = append(vcards, vcard)
	}

	var msg *waE2E.Message
	if len(vcards) == 1 {
		msg = &waE2E.Message{
			ContactMessage: &waE2E.ContactMessage{
				DisplayName: proto.String(opts.Contacts[0].FullName),
				Vcard:       proto.String(vcards[0]),
			},
		}
	} else {
		combined := ""
		for _, v := range vcards {
			combined += v + "\n"
		}
		msg = &waE2E.Message{
			ContactMessage: &waE2E.ContactMessage{
				DisplayName: proto.String("Contacts"),
				Vcard:       proto.String(combined),
			},
		}
	}

	resp, err := w.client.SendMessage(context.Background(), toJID(opts.To), msg)
	if err != nil {
		return SendResult{}, err
	}
	return SendResult{MessageID: resp.ID, Provider: "whatsmeow", Timestamp: time.Now().UnixMilli()}, nil
}

func (w *WhatsmeowProvider) SendReaction(opts SendReactionOptions) (SendResult, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return SendResult{}, fmt.Errorf("not connected")
	}

	resp, err := w.client.SendMessage(context.Background(), toJID(opts.To), w.client.BuildReaction(toJID(opts.To), types.EmptyJID, opts.MessageID, opts.Emoji))
	if err != nil {
		return SendResult{}, err
	}
	return SendResult{MessageID: resp.ID, Provider: "whatsmeow", Timestamp: time.Now().UnixMilli()}, nil
}

func (w *WhatsmeowProvider) SendPoll(opts SendPollOptions) (SendResult, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return SendResult{}, fmt.Errorf("not connected")
	}

	resp, err := w.client.SendMessage(context.Background(), toJID(opts.To), w.client.BuildPollCreation(opts.Name, opts.Options, opts.SelectableCount))
	if err != nil {
		return SendResult{}, err
	}
	return SendResult{MessageID: resp.ID, Provider: "whatsmeow", Timestamp: time.Now().UnixMilli()}, nil
}

func (w *WhatsmeowProvider) CheckNumbers(numbers []string) ([]NumberCheck, error) {
	if w.client == nil || w.Status() != StatusConnected {
		return nil, fmt.Errorf("not connected")
	}

	results := make([]NumberCheck, 0, len(numbers))
	for _, num := range numbers {
		jid := toJID(num)
		resp, err := w.client.IsOnWhatsApp(context.Background(), []string{"+" + jid.User})
		if err != nil {
			results = append(results, NumberCheck{Number: num, Exists: false})
			continue
		}
		for _, r := range resp {
			results = append(results, NumberCheck{
				Number: num,
				Exists: r.IsIn,
				JID:    r.JID.String(),
			})
		}
	}
	return results, nil
}

func (w *WhatsmeowProvider) UpdateProfileName(name string) error {
	if w.client == nil || w.Status() != StatusConnected {
		return fmt.Errorf("not connected")
	}
	// Nome de exibicao e configuracao de app state. Antes chamava SetStatusMessage, que troca
	// o recado ("about") e deixava o nome igual.
	return w.client.SendAppState(context.Background(), appstate.BuildSettingPushName(name))
}

func (w *WhatsmeowProvider) UpdateProfileStatus(status string) error {
	if w.client == nil || w.Status() != StatusConnected {
		return fmt.Errorf("not connected")
	}
	return w.client.SetStatusMessage(context.Background(), types.SetStatusInput{Text: &status})
}

func (w *WhatsmeowProvider) UpdateProfilePicture(base64Image string) error {
	if w.client == nil || w.Status() != StatusConnected {
		return fmt.Errorf("not connected")
	}
	data, err := media.FromBase64(base64Image)
	if err != nil {
		return fmt.Errorf("invalid base64: %w", err)
	}
	_, err = w.client.SetGroupPhoto(context.Background(), w.client.Store.ID.ToNonAD(), data)
	return err
}

func (w *WhatsmeowProvider) PairPhone(phone string) (string, error) {
	if w.client == nil {
		return "", fmt.Errorf("not initialized")
	}
	code, err := w.client.PairPhone(context.Background(), phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", err
	}
	return code, nil
}

// sqliteDSN abre o arquivo da sessao no driver modernc (Go puro). Chaves estrangeiras sao
// exigidas pela whatsmeow; busy_timeout evita "database is locked" quando o envio e a
// recepcao gravam ao mesmo tempo; WAL deixa leitura e escrita andarem juntas.
func sqliteDSN(path string) string {
	return "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
}
