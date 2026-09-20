package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

// Chain membungkus handler dengan beberapa middleware sekaligus.
//
// Urutannya dari luar ke dalam: Chain(h, A, B) menghasilkan A(B(h)), jadi A
// melihat request lebih dulu. Ini urutan yang sama seperti membacanya dari
// atas ke bawah.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
