// Copyright (C) 2025 The go-sqlite3 contributors.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build sqlite_spellfix1 || spellfix1
// +build sqlite_spellfix1 spellfix1

package sqlite3

/*
#cgo CFLAGS: -DSQLITE_ENABLE_FTS3 -DSQLITE_ENABLE_FTS3_PARENTHESIS -DSQLITE_ENABLE_FTS4_UNICODE61

#ifndef USE_LIBSQLITE3
#include "sqlite3-binding.h"
#else
#include <sqlite3.h>
#endif
#include <sqlite3ext.h>

// Declare spellfix1 initializer as a weak symbol so builds succeed
// even if the object providing it is not linked. When present, the
// extension will be auto-registered for all connections.
extern int sqlite3_spellfix1_init(sqlite3*, char**, const sqlite3_api_routines*) __attribute__((weak));

static void register_spellfix1() {
  if (sqlite3_spellfix1_init) {
    sqlite3_auto_extension((void(*)(void))sqlite3_spellfix1_init);
  }
}
*/
import "C"

func init() { C.register_spellfix1() }


