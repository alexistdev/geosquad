package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Chain(h, A, B) harus menghasilkan A(B(h)): A melihat request lebih dulu.
func TestChainOrder(t *testing.T) {
	var order []string

	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	final := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	})

	Chain(final, mark("pertama"), mark("kedua")).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"pertama", "kedua", "handler"}
	for i := range want {
		if i >= len(order) || order[i] != want[i] {
			t.Fatalf("urutan = %v, mau %v", order, want)
		}
	}
}

func TestRecoverTurnsPanicInto500(t *testing.T) {
	h := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sesuatu meledak")
	}))

	rec := httptest.NewRecorder()
	// Kalau Recover tidak bekerja, panic ini akan menjatuhkan test.
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, mau 500", rec.Code)
	}
}

func TestCORSOnlyReflectsAllowedOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:4200"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
	)

	cases := []struct {
		origin string
		want   string
	}{
		{"http://localhost:4200", "http://localhost:4200"},
		{"http://jahat.example.com", ""},
		{"", ""},
	}

	for _, c := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != c.want {
			t.Errorf("origin %q -> Allow-Origin %q, mau %q", c.origin, got, c.want)
		}
	}
}

func TestCORSPreflightShortCircuits(t *testing.T) {
	reached := false
	h := CORS([]string{"http://localhost:4200"})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/runs", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	h.ServeHTTP(rec, req)

	if reached {
		t.Error("preflight diteruskan ke handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, mau 204", rec.Code)
	}
}

// RequireRole dipasang setelah Authenticate. Kalau dipasang sendirian,
// request harus ditolak 401, bukan lolos diam-diam.
func TestRequireRoleWithoutActorReturns401(t *testing.T) {
	h := RequireRole("ADMIN")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler tidak boleh tercapai")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, mau 401", rec.Code)
	}
}

// statusWriter harus tetap meneruskan Flush, kalau tidak SSE berhenti bekerja
// begitu handler-nya dibungkus RequestLogger.
func TestStatusWriterForwardsFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec}

	if _, ok := any(sw).(http.Flusher); !ok {
		t.Fatal("statusWriter tidak mengimplementasikan http.Flusher")
	}
	sw.Flush()
	if !rec.Flushed {
		t.Error("Flush tidak diteruskan ke ResponseWriter asli")
	}
}
