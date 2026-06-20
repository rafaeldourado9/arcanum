package media

import "encoding/base64"

func ToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func FromBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
