package provider

type ConnectionStatus string

const (
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusQR           ConnectionStatus = "qr"
	StatusConnected    ConnectionStatus = "connected"
)

type SendTextOptions struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

type SendMediaOptions struct {
	To       string `json:"to"`
	Type     string `json:"type"`
	Data     []byte `json:"-"`
	URL      string `json:"url,omitempty"`
	MimeType string `json:"mimetype"`
	Filename string `json:"filename,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type SendResult struct {
	MessageID string `json:"messageId"`
	Provider  string `json:"provider"`
	Timestamp int64  `json:"timestamp"`
}

type MediaData struct {
	MimeType string `json:"mimetype"`
	Data     []byte `json:"-"`
	DataB64  string `json:"data,omitempty"`
	Filename string `json:"filename,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type IncomingMessage struct {
	MessageID string     `json:"messageId"`
	From      string     `json:"from"`
	Timestamp int64      `json:"timestamp"`
	PushName  string     `json:"pushName,omitempty"`
	Type      string     `json:"type"`
	Text      string     `json:"text,omitempty"`
	Media     *MediaData `json:"media,omitempty"`
}

type MessageHandler func(msg IncomingMessage)

type WhatsAppProvider interface {
	Name() string
	Connect() error
	Disconnect() error
	Status() ConnectionStatus
	QRCode() string

	SendText(opts SendTextOptions) (SendResult, error)
	SendMedia(opts SendMediaOptions) (SendResult, error)
	SendPresence(to string, presenceType string) error
	MarkAsRead(messageID string, from string) error

	OnMessage(handler MessageHandler)

	SendLocation(opts SendLocationOptions) (SendResult, error)
	SendContact(opts SendContactOptions) (SendResult, error)
	SendReaction(opts SendReactionOptions) (SendResult, error)
	SendPoll(opts SendPollOptions) (SendResult, error)

	CheckNumbers(numbers []string) ([]NumberCheck, error)
	UpdateProfileName(name string) error
	UpdateProfileStatus(status string) error
	UpdateProfilePicture(base64Image string) error

	PairPhone(phone string) (string, error)
}

type NumberCheck struct {
	Number string `json:"number"`
	Exists bool   `json:"exists"`
	JID    string `json:"jid,omitempty"`
}

type SendLocationOptions struct {
	To        string  `json:"to"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type SendReactionOptions struct {
	To        string `json:"to"`
	MessageID string `json:"messageId"`
	Emoji     string `json:"emoji"`
}

type ContactCard struct {
	FullName    string `json:"fullName"`
	PhoneNumber string `json:"phoneNumber"`
}

type SendContactOptions struct {
	To       string        `json:"to"`
	Contacts []ContactCard `json:"contacts"`
}

type SendPollOptions struct {
	To              string   `json:"to"`
	Name            string   `json:"name"`
	Options         []string `json:"options"`
	SelectableCount int      `json:"selectableCount"`
}
