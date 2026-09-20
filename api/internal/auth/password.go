package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Cost 10 adalah default bcrypt: cukup lambat untuk menyulitkan brute force,
// masih cukup cepat untuk login interaktif.
const bcryptCost = 10

func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hashed), nil
}

func CheckPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
