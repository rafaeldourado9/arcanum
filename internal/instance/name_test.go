package instance

import (
	"strings"
	"testing"
)

func TestValidName(t *testing.T) {
	ok := []string{"vendas", "central-8266fa731dbb4168", "loja_01", "A", strings.Repeat("a", 64)}
	bad := []string{"", "../etc", "a/b", `a\b`, "a b", "a.db", "..", "x;rm", strings.Repeat("a", 65), "ção"}
	for _, n := range ok {
		if !ValidName(n) {
			t.Errorf("ValidName(%q) = false, want true", n)
		}
	}
	for _, n := range bad {
		if ValidName(n) {
			t.Errorf("ValidName(%q) = true, want false", n)
		}
	}
}
