package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateURL(t *testing.T) {
	ok := []string{
		"https://example.com/hook",
		"http://api:8000/api/webhooks/qrcode/x", // outro container na rede do Docker
		"http://10.0.0.5/hook",
		"http://192.168.1.10:8080/hook",
	}
	bad := []string{
		"", "ftp://example.com", "file:///etc/passwd", "javascript:alert(1)",
		"http://169.254.169.254/latest/meta-data/", "http://[fd00:ec2::254]/",
		"http://0.0.0.0/", "http://user:pass@example.com/", "https:///nohost",
	}
	for _, u := range ok {
		if err := ValidateURL(u); err != nil {
			t.Errorf("ValidateURL(%q) = %v, want nil", u, err)
		}
	}
	for _, u := range bad {
		if err := ValidateURL(u); err == nil {
			t.Errorf("ValidateURL(%q) = nil, want error", u)
		}
	}
}

func TestSafeClientDoesNotFollowRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("redirect was followed")
	}))
	defer target.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redir.Close()

	resp, err := safeClient(5*time.Second).Post(redir.URL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302 (not followed)", resp.StatusCode)
	}
}
