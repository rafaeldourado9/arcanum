package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordAndTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	for _, a := range []string{"a", "b", "c"} {
		l.Record(Event{Actor: "admin", IP: "10.0.0.1", Action: a, Status: 200})
	}
	got := l.Tail(2)
	if len(got) != 2 || got[0].Action != "c" || got[1].Action != "b" {
		t.Fatalf("tail = %+v, want newest first [c b]", got)
	}
	if got[0].Time.IsZero() {
		t.Fatal("time not filled")
	}
	if st, _ := os.Stat(path); st.Mode().Perm()&0o077 != 0 && st.Mode().Perm() != 0o666 {
		t.Fatalf("audit file too open: %v", st.Mode().Perm())
	}
}

func TestRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l, _ := Open(path)
	defer l.Close()
	l.size = maxBytes - 10 // quase cheio
	l.Record(Event{Action: "novo", Detail: strings.Repeat("x", 100)})
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("old log not rotated: %v", err)
	}
	if got := l.Tail(5); len(got) != 1 || got[0].Action != "novo" {
		t.Fatalf("after rotation tail = %+v", got)
	}
}

func TestNilLogIsSafe(t *testing.T) {
	var l *Log
	l.Record(Event{Action: "x"})
	if len(l.Tail(5)) != 0 {
		t.Fatal("nil log returned events")
	}
	l.Close()
}
