package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct {
	cost int
}

func NewBcrypt(cost int) (Bcrypt, error) {
	return Bcrypt{
		cost: cost,
	}, nil
}

func (b *Bcrypt) Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), b.cost)

	return string(hash), fmt.Errorf("bcrypt hash failed: %w", err)
}

func (b *Bcrypt) Compare(hashed string, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	if err != nil {
		return fmt.Errorf("bcrypt compare failed: %w", err)
	}

	return nil
}
