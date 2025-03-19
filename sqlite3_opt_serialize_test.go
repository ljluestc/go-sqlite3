//go:build !libsqlite3 || sqlite_serialize
// +build !libsqlite3 sqlite_serialize

package sqlite3

import (
    "context"
    "database/sql"
    "os"
    "testing"
)

const testDriverName = "sqlite3"

func TempFilenameSerialize(tb testing.TB) string {
    tb.Helper()
    f, err := os.CreateTemp("", "sqlite-serialize-test-*.db")
    if err != nil {
        tb.Fatalf("Failed to create temp file: %v", err)
    }
    defer f.Close()
    return f.Name()
}

func TestSerializeDeserialize(t *testing.T) {
    t.Run("BasicSerializeDeserialize", func(t *testing.T) {
        srcTempFilename := TempFilenameSerialize(t)
        defer os.Remove(srcTempFilename)
        srcDb, err := sql.Open(testDriverName, srcTempFilename)
        if err != nil {
            t.Fatal("Failed to open the source database:", err)
        }
        defer srcDb.Close()
        if err = srcDb.Ping(); err != nil {
            t.Fatal("Failed to connect to the source database:", err)
        }

        destTempFilename := TempFilenameSerialize(t)
        defer os.Remove(destTempFilename)
        destDb, err := sql.Open(testDriverName, destTempFilename)
        if err != nil {
            t.Fatal("Failed to open the destination database:", err)
        }
        defer destDb.Close()
        if err = destDb.Ping(); err != nil {
            t.Fatal("Failed to connect to the destination database:", err)
        }

        _, err = srcDb.Exec(`CREATE TABLE foo (name TEXT)`)
        if err != nil {
            t.Fatal("Failed to create table in source database:", err)
        }
        _, err = srcDb.Exec(`INSERT INTO foo(name) VALUES('alice')`)
        if err != nil {
            t.Fatal("Failed to insert data into source database:", err)
        }

        srcConn, err := srcDb.Conn(context.Background())
        if err != nil {
            t.Fatal("Failed to get connection to source database:", err)
        }
        defer srcConn.Close()

        var serialized []byte
        if err := srcConn.Raw(func(raw any) error {
            var err error
            serialized, err = raw.(*SQLiteConn).Serialize("")
            return err
        }); err != nil {
            t.Fatal("Failed to serialize source database:", err)
        }

        if len(serialized) == 0 {
            t.Fatal("Serialized data is empty")
        }

        var destTableCount int
        err = destDb.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'").Scan(&destTableCount)
        if err != nil {
            t.Fatal("Failed to check the destination table count:", err)
        }
        if destTableCount != 0 {
            t.Fatalf("The destination database is not empty; %v table(s) found.", destTableCount)
        }

        destConn, err := destDb.Conn(context.Background())
        if err != nil {
            t.Fatal("Failed to get connection to destination database:", err)
        }
        defer destConn.Close()

        if err := destConn.Raw(func(raw any) error {
            return raw.(*SQLiteConn).Deserialize(serialized, "")
        }); err != nil {
            t.Fatal("Failed to deserialize source database:", err)
        }

        var destRowCount int
        err = destDb.QueryRow(`SELECT COUNT(*) FROM foo`).Scan(&destRowCount)
        if err != nil {
            t.Fatal("Failed to count rows in destination database table:", err)
        }
        if destRowCount != 1 {
            t.Fatalf("Expected 1 row in destination table, got %d", destRowCount)
        }

        var name string
        err = destDb.QueryRow(`SELECT name FROM foo`).Scan(&name)
        if err != nil {
            t.Fatal("Failed to query name from destination database:", err)
        }
        if name != "alice" {
            t.Fatalf("Expected name 'alice', got '%s'", name)
        }
    })

    t.Run("EmptyDataDeserialize", func(t *testing.T) {
        tempFilename := TempFilenameSerialize(t)
        defer os.Remove(tempFilename)
        db, err := sql.Open(testDriverName, tempFilename)
        if err != nil {
            t.Fatal("Failed to open database:", err)
        }
        defer db.Close()

        conn, err := db.Conn(context.Background())
        if err != nil {
            t.Fatal("Failed to get connection:", err)
        }
        defer conn.Close()

        err = conn.Raw(func(raw any) error {
            return raw.(*SQLiteConn).Deserialize([]byte{}, "")
        })
        if err == nil {
            t.Fatal("Expected error when deserializing empty data, got nil")
        }
        if err.Error() != "cannot deserialize empty data" {
            t.Fatalf("Expected 'cannot deserialize empty data' error, got: %v", err)
        }
    })

    t.Run("CustomSchema", func(t *testing.T) {
        srcTempFilename := TempFilenameSerialize(t)
        defer os.Remove(srcTempFilename)
        srcDb, err := sql.Open(testDriverName, srcTempFilename)
        if err != nil {
            t.Fatal("Failed to open source database:", err)
        }
        defer srcDb.Close()

        _, err = srcDb.Exec(`ATTACH DATABASE ':memory:' AS custom`)
        if err != nil {
            t.Fatal("Failed to attach custom schema:", err)
        }
        _, err = srcDb.Exec(`CREATE TABLE custom.foo (name TEXT)`)
        if err != nil {
            t.Fatal("Failed to create table in custom schema:", err)
        }
        _, err = srcDb.Exec(`INSERT INTO custom.foo(name) VALUES('bob')`)
        if err != nil {
            t.Fatal("Failed to insert into custom schema:", err)
        }

        srcConn, err := srcDb.Conn(context.Background())
        if err != nil {
            t.Fatal("Failed to get source connection:", err)
        }
        defer srcConn.Close()

        var serialized []byte
        if err := srcConn.Raw(func(raw any) error {
            var err error
            serialized, err = raw.(*SQLiteConn).Serialize("custom")
            return err
        }); err != nil {
            t.Fatal("Failed to serialize custom schema:", err)
        }

        destTempFilename := TempFilenameSerialize(t)
        defer os.Remove(destTempFilename)
        destDb, err := sql.Open(testDriverName, destTempFilename)
        if err != nil {
            t.Fatal("Failed to open destination database:", err)
        }
        defer destDb.Close()

        destConn, err := destDb.Conn(context.Background())
        if err != nil {
            t.Fatal("Failed to get destination connection:", err)
        }
        defer destConn.Close()

        if err := destConn.Raw(func(raw any) error {
            return raw.(*SQLiteConn).Deserialize(serialized, "custom")
        }); err != nil {
            t.Fatal("Failed to deserialize custom schema:", err)
        }

        var name string
        err = destDb.QueryRow(`SELECT name FROM custom.foo`).Scan(&name)
        if err != nil {
            t.Fatal("Failed to query custom schema:", err)
        }
        if name != "bob" {
            t.Fatalf("Expected name 'bob' in custom schema, got '%s'", name)
        }
    })
}