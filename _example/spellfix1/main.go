package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    "github.com/mattn/go-sqlite3"
)

// Example program that loads the spellfix1 extension and performs a fuzzy match.
//
// Build the shared extension first (see Makefile in this directory), then run:
//   SPELLFIX1_SO=./spellfix1.so go run .
func main() {
    so := os.Getenv("SPELLFIX1_SO")
    if so == "" {
        log.Fatal("set SPELLFIX1_SO to the path of spellfix1 shared library (e.g. ./spellfix1.so)")
    }

    sql.Register("sqlite3_with_spellfix1", &sqlite3.SQLiteDriver{Extensions: []string{so}})
    db, err := sql.Open("sqlite3_with_spellfix1", ":memory:")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    if _, err := db.Exec("CREATE VIRTUAL TABLE s USING spellfix1"); err != nil {
        log.Fatal(err)
    }
    if _, err := db.Exec("INSERT INTO s(word) VALUES ('amsterdam'), ('amstel'), ('rotterdam')"); err != nil {
        log.Fatal(err)
    }
    row := db.QueryRow("SELECT word, distance FROM s WHERE word MATCH 'amsterdm' ORDER BY distance, word LIMIT 1")
    var word string
    var dist int
    if err := row.Scan(&word, &dist); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Best match: %s (distance=%d)\n", word, dist)
}


