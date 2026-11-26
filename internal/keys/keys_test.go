package keys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAuthenticateRevoke(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	secret, k, err := s.Create("agente-ia", []string{"loja", "loja", "suporte"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, Prefix) || len(secret) != len(Prefix)+64 {
		t.Fatalf("unexpected secret shape %q", secret)
	}
	if len(k.Instances) != 2 || !k.ReadOnly || !k.Allows("loja") || k.Allows("outra") {
		t.Fatalf("bad key: %+v", k)
	}

	got, ok := s.Authenticate(secret)
	if !ok || got.ID != k.ID {
		t.Fatal("valid secret was not accepted")
	}
	if _, ok := s.Authenticate(secret + "x"); ok {
		t.Fatal("wrong secret accepted")
	}
	if _, ok := s.Authenticate("sem-prefixo"); ok {
		t.Fatal("secret without prefix accepted")
	}

	// O arquivo nao guarda o segredo, so o hash, e e privado.
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), secret) {
		t.Fatal("secret was written to disk")
	}
	if st, _ := os.Stat(path); st.Mode().Perm()&0o077 != 0 && st.Mode().Perm() != 0o666 {
		t.Fatalf("keys file too open: %v", st.Mode().Perm())
	}

	// Sobrevive a reinicio.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.Authenticate(secret); !ok {
		t.Fatal("key lost after reopen")
	}

	if _, err := s2.Revoke(k.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.Authenticate(secret); ok {
		t.Fatal("revoked key still works")
	}
	if l := s2.List(); len(l) != 1 || !l[0].Revoked {
		t.Fatalf("revoked key should stay listed: %+v", l)
	}
	if _, err := s2.Revoke("key_inexistente"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateValidation(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "keys.json"))
	for name, tc := range map[string]struct {
		name      string
		instances []string
	}{
		"sem nome":       {"", []string{"a"}},
		"sem sessoes":    {"x", nil},
		"nome invalido":  {"x", []string{"../etc"}},
		"nome com ponto": {"x", []string{"a.b"}},
	} {
		if _, _, err := s.Create(tc.name, tc.instances, false); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestNilStoreIsSafe(t *testing.T) {
	var s *Store
	if _, ok := s.Authenticate("arc_x"); ok {
		t.Fatal("nil store authenticated")
	}
	if len(s.List()) != 0 {
		t.Fatal("nil store listed keys")
	}
}
