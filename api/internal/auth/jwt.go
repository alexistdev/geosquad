// Package auth memegang penerbitan JWT dan penyimpanan sesi di Redis.
//
// Polanya mengikuti project lain (geolicense): JWT dibuat saat login, tapi
// tidak pernah dikirim ke browser. Yang dikirim adalah sessionId acak, dan
// JWT-nya disimpan di Redis dengan sessionId sebagai kunci. Efeknya sesi bisa
// dicabut seketika dengan menghapus satu key, sesuatu yang tidak bisa
// dilakukan JWT murni.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/alexistdev/geosquad/api/internal/config"
	"github.com/alexistdev/geosquad/api/internal/model"
)

var ErrInvalidToken = errors.New("token tidak valid")

type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

func NewTokenService(cfg config.JWTConfig, issuer string) *TokenService {
	return &TokenService{
		secret:     []byte(cfg.Secret),
		expiration: cfg.Expiration,
		issuer:     issuer,
	}
}

func (s *TokenService) Generate(u *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   string(u.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Email,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			ID:        newID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("tanda tangani jwt: %w", err)
	}
	return signed, nil
}

// Parse memverifikasi tanda tangan dan masa berlaku token.
func (s *TokenService) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		// Algoritma dikunci ke HMAC. Tanpa cek ini, token ber-alg "none" atau
		// RS256 dengan public key sebagai secret bisa lolos.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritma tidak didukung: %v", t.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims.UserID == "" {
		return nil, fmt.Errorf("%w: klaim uid kosong", ErrInvalidToken)
	}
	return claims, nil
}

func (s *TokenService) ExpiresIn() int64 {
	return int64(s.expiration.Seconds())
}
