package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/alexistdev/geosquad/api/internal/cache"
)

var ErrSessionNotFound = errors.New("sesi tidak ditemukan atau sudah kedaluwarsa")

const (
	sessionPrefix   = "geosquad:session:"
	userIndexPrefix = "geosquad:user-sessions:"
)

type SessionStore struct {
	rdb *cache.Client
	ttl time.Duration
}

func NewSessionStore(rdb *cache.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func sessionKey(id string) string    { return sessionPrefix + id }
func userIndexKey(uid string) string { return userIndexPrefix + uid }

// Create menyimpan JWT di Redis dan mengembalikan sessionId yang akan dipegang
// client. Selain key sesi, id-nya juga dicatat di sebuah set per user supaya
// "logout semua perangkat" dan pencabutan saat user disuspend bisa dilakukan
// tanpa memindai seluruh keyspace Redis.
func (s *SessionStore) Create(ctx context.Context, userID, token string) (string, error) {
	sessionID := newID()

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, sessionKey(sessionID), token, s.ttl)
	pipe.SAdd(ctx, userIndexKey(userID), sessionID)
	// Index diberi TTL sedikit lebih panjang supaya tidak kedaluwarsa lebih
	// dulu daripada sesi yang didaftarkannya.
	pipe.Expire(ctx, userIndexKey(userID), s.ttl+time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("simpan sesi: %w", err)
	}
	return sessionID, nil
}

// Resolve menukar sessionId dengan JWT yang tersimpan.
func (s *SessionStore) Resolve(ctx context.Context, sessionID string) (string, error) {
	token, err := s.rdb.Get(ctx, sessionKey(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("baca sesi: %w", err)
	}
	return token, nil
}

// Refresh memperpanjang umur sesi yang sedang dipakai.
func (s *SessionStore) Refresh(ctx context.Context, sessionID string) error {
	ok, err := s.rdb.Expire(ctx, sessionKey(sessionID), s.ttl).Result()
	if err != nil {
		return fmt.Errorf("perpanjang sesi: %w", err)
	}
	if !ok {
		return ErrSessionNotFound
	}
	return nil
}

func (s *SessionStore) Destroy(ctx context.Context, userID, sessionID string) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey(sessionID))
	pipe.SRem(ctx, userIndexKey(userID), sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("hapus sesi: %w", err)
	}
	return nil
}

// DestroyAll mencabut semua sesi milik satu user. Dipanggil saat user
// disuspend, dihapus, atau ganti password.
func (s *SessionStore) DestroyAll(ctx context.Context, userID string) error {
	ids, err := s.rdb.SMembers(ctx, userIndexKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("baca daftar sesi user: %w", err)
	}

	pipe := s.rdb.TxPipeline()
	for _, id := range ids {
		pipe.Del(ctx, sessionKey(id))
	}
	pipe.Del(ctx, userIndexKey(userID))
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("hapus semua sesi: %w", err)
	}
	return nil
}

// newID menghasilkan identifier acak 256 bit. UUIDv4 dipakai untuk id domain,
// tapi untuk session id yang jadi penentu akses, entropi penuh lebih pantas.
func newID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand gagal berarti sistemnya bermasalah serius. Lebih baik
		// jatuh ke UUIDv4 daripada mengembalikan string kosong yang bisa
		// dipakai sebagai session id.
		return uuid.NewString()
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
