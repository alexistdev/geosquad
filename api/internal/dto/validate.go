package dto

import (
	"regexp"
	"unicode/utf8"
)

// Cukup untuk menyaring salah ketik. Validasi email yang sungguhan hanya bisa
// dilakukan dengan mengirim email verifikasi.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // batas keras bcrypt: byte ke-73 dan seterusnya diabaikan
)

func validateEmail(email string) []string {
	var errs []string
	switch {
	case email == "":
		errs = append(errs, "Email wajib diisi.")
	case len(email) > 190:
		errs = append(errs, "Email maksimal 190 karakter.")
	case !emailPattern.MatchString(email):
		errs = append(errs, "Format email tidak valid.")
	}
	return errs
}

func validatePassword(pw string) []string {
	var errs []string
	switch {
	case pw == "":
		errs = append(errs, "Password wajib diisi.")
	case utf8.RuneCountInString(pw) < minPasswordLen:
		errs = append(errs, "Password minimal 8 karakter.")
	case len(pw) > maxPasswordLen:
		// Dibatasi eksplisit, bukan dibiarkan terpotong diam-diam oleh bcrypt.
		errs = append(errs, "Password maksimal 72 byte.")
	}
	return errs
}
