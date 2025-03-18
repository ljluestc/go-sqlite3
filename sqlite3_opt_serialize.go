//go:build !libsqlite3 || sqlite_serialize
// +build !libsqlite3 sqlite_serialize

package sqlite3

import (
    "fmt"
    "math"
    "reflect"
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

// Serialize returns a byte slice that is a serialization of the database.
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

    cBuf := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
        Data: uintptr(unsafe.Pointer(ptr)),
        Len:  int(sz),
        Cap:  int(sz),
    }))

    res := make([]byte, int(sz))
    copy(res, cBuf)
    return res, nil
}

// Deserialize causes the connection to disconnect from the current database and
// re-open as an in-memory database based on the contents of the byte slice.
func (c *SQLiteConn) Deserialize(b []byte, schema string) error {
    if schema == "" {
        schema = "main"
    }
    if len(b) == 0 {
        return fmt.Errorf("cannot deserialize empty data")
    }

    zSchema := C.CString(schema)
    defer C.free(unsafe.Pointer(zSchema))

    // Allocate C memory and copy data
    tmpBuf := (*C.uchar)(C.sqlite3_malloc64(C.sqlite3_uint64(len(b))))
    if tmpBuf == nil {
        return fmt.Errorf("failed to allocate memory for deserialization")
    }
    // Copy Go slice to C memory without using reflect.SliceHeader
    for i := 0; i < len(b); i++ {
        *(*C.uchar)(unsafe.Pointer(uintptr(unsafe.Pointer(tmpBuf)) + uintptr(i))) = C.uchar(b[i])
    }

    // Perform deserialization
    rc := C.sqlite3_deserialize(c.db, zSchema, tmpBuf, C.sqlite3_int64(len(b)),
        C.sqlite3_int64(len(b)), C.SQLITE_DESERIALIZE_FREEONCLOSE|C.SQLITE_DESERIALIZE_RESIZEABLE)
    if rc != C.SQLITE_OK {
        C.sqlite3_free(unsafe.Pointer(tmpBuf)) // Free only if deserialization fails
        return fmt.Errorf("deserialize failed for schema %q with return code %d: %w", schema, rc, c.lastError())
    }
    // Note: tmpBuf is freed by SQLite via FREEONCLOSE on success
    return nil
}