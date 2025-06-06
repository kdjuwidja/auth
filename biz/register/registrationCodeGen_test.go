package bizregister

import (
	"testing"
)

func TestGenerateRegistrationCode(t *testing.T) {
	// Test multiple generations
	for i := 0; i < 1000; i++ {
		code, err := generateRegistrationCode()
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("Expected code length 6, got %d", len(code))
		}

		// Test character set
		for _, c := range code {
			if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				t.Errorf("Invalid character in code: %c", c)
			}
		}
	}
}
