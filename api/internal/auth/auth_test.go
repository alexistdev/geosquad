package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/alexistdev/geosquad/api/internal/config"
	"github.com/alexistdev/geosquad/api/internal/model"
)

func testTokens(t *testing.T) *TokenService {
	t.Helper()
	return NewTokenService(config.JWTConfig{
		Secret:     strings.Repeat("x", 48),
		Expiration: time.Hour,
	}, "geosquad-test")
}

func sampleUser() *model.User {
	return &model.User{
		ID:    "11111111-1111-1111-1111-111111111111",
		Email: "budi@example.com",
		Role:  model.RoleAdmin,
	}
}

func TestPasswordRoundTrip(t *testing.T) {
	hashed, err := HashPassword("rahasia-banget")
	if err != nil {
		t.Fatalf("hash gagal: %v", err)
	}
	if hashed == "rahasia-banget" {
		t.Fatal("password tersimpan sebagai teks polos")
	}
	if !CheckPassword(hashed, "rahasia-banget") {
		t.Error("password yang benar ditolak")
	}
	if CheckPassword(hashed, "rahasia-bangeT") {
		t.Error("password yang salah diterima")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	svc := testTokens(t)
	user := sampleUser()

	token, err := svc.Generate(user)
	if err != nil {
		t.Fatalf("generate gagal: %v", err)
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("parse gagal: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("uid = %q, mau %q", claims.UserID, user.ID)
	}
	if claims.Subject != user.Email {
		t.Errorf("subject = %q, mau %q", claims.Subject, user.Email)
	}
}

// Token yang ditandatangani secret lain harus ditolak. Ini inti dari kenapa
// JWT bisa dipercaya sama sekali.
func TestTokenRejectsForeignSecret(t *testing.T) {
	issuer := testTokens(t)
	token, err := issuer.Generate(sampleUser())
	if err != nil {
		t.Fatalf("generate gagal: %v", err)
	}

	other := NewTokenService(config.JWTConfig{
		Secret:     strings.Repeat("y", 48),
		Expiration: time.Hour,
	}, "geosquad-test")

	if _, err := other.Parse(token); err == nil {
		t.Error("token dari secret lain diterima")
	}
}

func TestTokenRejectsExpired(t *testing.T) {
	svc := NewTokenService(config.JWTConfig{
		Secret:     strings.Repeat("x", 48),
		Expiration: -time.Minute, // sudah lewat saat diterbitkan
	}, "geosquad-test")

	token, err := svc.Generate(sampleUser())
	if err != nil {
		t.Fatalf("generate gagal: %v", err)
	}
	if _, err := svc.Parse(token); err == nil {
		t.Error("token kedaluwarsa diterima")
	}
}

// Serangan alg=none: penyerang membuang tanda tangan dan mengubah header.
// Tanpa penguncian metode, parser bisa menerimanya.
func TestTokenRejectsAlgNone(t *testing.T) {
	svc := testTokens(t)
	// header {"alg":"none","typ":"JWT"} . payload . (tanda tangan kosong)
	forged := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
		"eyJ1aWQiOiJoYWNrZXIiLCJzdWIiOiJoYWNrZXJAZXZpbC5jb20ifQ."

	if _, err := svc.Parse(forged); err == nil {
		t.Error("token alg=none diterima")
	}
}

func TestNewIDIsUniqueAndLongEnough(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := newID()
		if len(id) < 32 {
			t.Fatalf("session id terlalu pendek: %d karakter", len(id))
		}
		if _, dup := seen[id]; dup {
			t.Fatal("session id berulang")
		}
		seen[id] = struct{}{}
	}
}
