//go:build !sqlite_omit_load_extension
// +build !sqlite_omit_load_extension

package sqlite3

import (
    "database/sql"
    "os"
    "testing"
)

// TestSpellfix1Loadable verifies spellfix1 works when loaded as a shared extension (.so/.dylib/.dll).
//
// To run locally, first build the loadable extension and point SPELLFIX1_SO to it, e.g.:
//   SPELLFIX1_SO=/path/to/spellfix1.so go test -run TestSpellfix1Loadable
func TestSpellfix1Loadable(t *testing.T) {
    so := os.Getenv("SPELLFIX1_SO")
    if so == "" {
        t.Skip("SPELLFIX1_SO not set; skipping loadable spellfix1 test")
    }

    const drv = "sqlite3_with_spellfix1"
    sql.Register(drv, &SQLiteDriver{Extensions: []string{so}})

    db, err := sql.Open(drv, ":memory:")
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    defer db.Close()

    if _, err := db.Exec("CREATE VIRTUAL TABLE s USING spellfix1"); err != nil {
        t.Fatalf("create vtab: %v", err)
    }

    // A tiny dictionary
    if _, err := db.Exec("INSERT INTO s(word) VALUES ('amsterdam'), ('amstel'), ('rotterdam')"); err != nil {
        t.Fatalf("insert words: %v", err)
    }

    // Misspelled query should suggest 'amsterdam' with the smallest distance
    row := db.QueryRow("SELECT word FROM s WHERE word MATCH 'amsterdm' ORDER BY distance, word LIMIT 1")
    var got string
    if err := row.Scan(&got); err != nil {
        t.Fatalf("scan: %v", err)
    }
    want := "amsterdam"
    if got != want {
        t.Fatalf("want %q, got %q", want, got)
    }
}


