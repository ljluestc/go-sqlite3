//go:build !libsqlite3 || sqlite_serialize
// +build !libsqlite3 sqlite_serialize

package sqlite3

import (
    "fmt"
    "math"
    "unsafe"
)

/*
#ifndef USE_LIBSQLITE3
#include <sqlite3-binding.h>
#else
#include <sqlite3.h>
#endif
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

// SQLiteConn is a minimal representation of the SQLite connection for testing.
// In the full package, this is defined in sqlite3.go with additional fields.
type SQLiteConn struct {
    db *C.sqlite3 // Pointer to the SQLite database connection
}

// Serialize returns a byte slice that is a serialization of the database.
//
// See https://www.sqlite.org/c3ref/serialize.html
func (c *SQLiteConn) Serialize(schema string) ([]byte, error) {
    if schema == "" {
        schema = "main"
    }
    zSchema := C.CString(schema)
    defer C.free(unsafe.Pointer(zSchema))

    var sz C.sqlite3_int64
    ptr := C.sqlite3_serialize(c.db, zSchema, &sz, 0)
    if ptr == nil {
        return nil, fmt.Errorf("serialize failed for schema %q: %w", schema, c.lastError())
    }
    defer C.sqlite3_free(unsafe.Pointer(ptr))

    if sz < 0 || sz > C.sqlite3_int64(math.MaxInt) {
        return nil, fmt.Errorf("serialized database size invalid or too large (%d bytes)", sz)
    }

    // Replace reflect.SliceHeader with unsafe.Slice to fix unsafeptr
    cBuf := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(sz))
    // Copy to Go-managed memory to ensure safety after C.free
    res := make([]byte, int(sz))
    copy(res, cBuf)
    return res, nil
}

// Deserialize causes the connection to disconnect from the current database and
// re-open as an in-memory database based on the contents of the byte slice.
//
// See https://www.sqlite.org/c3ref/deserialize.html
func (c *SQLiteConn) Deserialize(b []byte, schema string) error {
    if schema == "" {
        schema = "main"
    }
    if len(b) == 0 {
        return fmt.Errorf("cannot deserialize empty data")
    }

    zSchema := C.CString(schema)
    defer C.free(unsafe.Pointer(zSchema))

    // Allocate C memory for the buffer
    tmpBuf := (*C.uchar)(C.sqlite3_malloc64(C.sqlite3_uint64(len(b))))
    if tmpBuf == nil {
        return fmt.Errorf("failed to allocate memory for deserialization")
    }
    defer C.sqlite3_free(unsafe.Pointer(tmpBuf)) // Free on failure or if not handed over

    // Replace reflect.SliceHeader with manual byte copy to fix unsafeptr
    for i := 0; i < len(b); i++ {
        *(*C.uchar)(unsafe.Pointer(uintptr(unsafe.Pointer(tmpBuf)) + uintptr(i))) = C.uchar(b[i])
    }

    // Perform deserialization
    rc := C.sqlite3_deserialize(c.db, zSchema, tmpBuf, C.sqlite3_int64(len(b)),
        C.sqlite3_int64(len(b)), C.SQLITE_DESERIALIZE_FREEONCLOSE|C.SQLITE_DESERIALIZE_RESIZEABLE)
    if rc != C.SQLITE_OK {
        return fmt.Errorf("deserialize failed for schema %q with return code %d: %w", schema, rc, c.lastError())
    }
    // On success, SQLite takes ownership of tmpBuf; no additional free needed
    return nil
}

// lastError retrieves the last error from the SQLite connection
func (c *SQLiteConn) lastError() error {
    if c.db == nil {
        return fmt.Errorf("no database connection")
    }
    errCode := C.sqlite3_errcode(c.db)
    errMsg := C.GoString(C.sqlite3_errmsg(c.db))
    return fmt.Errorf("sqlite error %d: %s", errCode, errMsg)
}