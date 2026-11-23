package provider

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// O driver SQLite em Go puro tem que abrir, migrar e reabrir o banco da sessao exatamente
// como a whatsmeow espera (chaves estrangeiras ligadas, WAL). Se o DSN ou o driver quebrar,
// nenhuma sessao conecta: este teste pega isso sem precisar de WhatsApp.
func TestSQLiteStoreOpensAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.db")
	ctx := context.Background()

	for round := 1; round <= 2; round++ {
		container, err := sqlstore.New(ctx, "sqlite", sqliteDSN(path), waLog.Noop)
		if err != nil {
			t.Fatalf("round %d: open store: %v", round, err)
		}
		device, err := container.GetFirstDevice(ctx)
		if err != nil {
			t.Fatalf("round %d: first device: %v", round, err)
		}
		if device == nil {
			t.Fatalf("round %d: nil device", round)
		}
		if err := container.Close(); err != nil {
			t.Fatalf("round %d: close: %v", round, err)
		}
	}

	// O mesmo DSN liga as chaves estrangeiras em toda conexao.
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var fk int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys = %d (%v), want 1", fk, err)
	}
}
