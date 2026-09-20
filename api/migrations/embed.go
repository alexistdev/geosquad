// Package migrations menanam file .sql ke dalam binary.
//
// Embed harus tinggal sekamar dengan file yang ditanamnya: go:embed menolak
// path yang mengandung "..", jadi deklarasinya tidak bisa diletakkan di
// internal/db.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

// Dir adalah root di dalam FS. File .sql ada di akar embed, bukan subfolder.
const Dir = "."
