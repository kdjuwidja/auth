package bizregister

import (
	"crypto/rand"
	"fmt"
)

func generateRegistrationCode() (string, error) {
	const (
		length = 6
		chars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	)

	code := make([]byte, length)

	_, err := rand.Read(code)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := 0; i < length; i++ {
		code[i] = chars[code[i]%byte(len(chars))]
	}

	return string(code), nil
}
