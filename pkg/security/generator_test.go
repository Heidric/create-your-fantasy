package security

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerateSecureString(t *testing.T) {
	tests := []struct {
		name   string
		length int
		valid  bool
	}{
		{"Valid minimum length", 4, true},
		{"Valid medium length", 8, true},
		{"Valid long length", 32, true},
		{"Invalid length - too short", 3, false},
		{"Invalid length - zero", 0, false},
		{"Invalid length - negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateSecureString(tt.length)

			if tt.valid {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				if len(result) != tt.length {
					t.Errorf("Expected length %d, got %d", tt.length, len(result))
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for length %d, got nil", tt.length)
				}
			}
		})
	}
}

func TestGenerateSecureStringContainsAllCategories(t *testing.T) {
	lengths := []int{4, 8, 12, 16, 32}

	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			result, err := GenerateSecureString(length)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			categories := map[string]bool{
				"uppercase": false,
				"lowercase": false,
				"digit":     false,
				"special":   false,
			}

			for _, char := range result {
				switch {
				case strings.ContainsRune(uppercaseLetters, char):
					categories["uppercase"] = true
				case strings.ContainsRune(lowercaseLetters, char):
					categories["lowercase"] = true
				case strings.ContainsRune(digits, char):
					categories["digit"] = true
				case strings.ContainsRune(specialChars, char):
					categories["special"] = true
				}
			}

			for category, found := range categories {
				if !found {
					t.Errorf("Missing %s character in result: %s", category, result)
				}
			}
		})
	}
}

func TestGenerateSecureStringNoAmbiguousCharacters(t *testing.T) {
	// Characters that should never appear in generated strings
	ambiguousChars := "0O1Il5Ss$6b8BNUVuvnm()[]|;:"

	// Generate multiple strings to increase confidence
	for i := 0; i < 100; i++ {
		result, err := GenerateSecureString(16)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		for _, ambiguous := range ambiguousChars {
			if strings.ContainsRune(result, ambiguous) {
				t.Errorf("Found ambiguous character '%c' in result: %s", ambiguous, result)
			}
		}
	}
}

func TestGenerateSecureStringRandomness(t *testing.T) {
	const iterations = 100
	const length = 12

	results := make(map[string]bool)

	// Generate multiple strings and check they're all different
	for i := 0; i < iterations; i++ {
		result, err := GenerateSecureString(length)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if results[result] {
			t.Errorf("Generated duplicate string: %s", result)
		}
		results[result] = true
	}

	// Should have generated unique strings
	if len(results) != iterations {
		t.Errorf("Expected %d unique strings, got %d", iterations, len(results))
	}
}

func TestGenerateSecureStringOnlyValidCharacters(t *testing.T) {
	allValidChars := uppercaseLetters + lowercaseLetters + digits + specialChars

	for i := 0; i < 50; i++ {
		result, err := GenerateSecureString(16)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		for _, char := range result {
			if !strings.ContainsRune(allValidChars, char) {
				t.Errorf("Found invalid character '%c' in result: %s", char, result)
			}
		}
	}
}

func TestGetRandomChar(t *testing.T) {
	tests := []struct {
		name  string
		chars string
		valid bool
	}{
		{"Valid string", "ABC123", true},
		{"Single character", "X", true},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char, err := getRandomChar(tt.chars)

			if tt.valid {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				if !strings.ContainsRune(tt.chars, rune(char)) {
					t.Errorf("Character '%c' not found in source string '%s'", char, tt.chars)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for empty string, got nil")
				}
			}
		})
	}
}

func TestGetRandomCharDistribution(t *testing.T) {
	const chars = "ABC"
	const iterations = 3000
	counts := make(map[byte]int)

	// Generate many random characters
	for i := 0; i < iterations; i++ {
		char, err := getRandomChar(chars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		counts[char]++
	}

	// Check that all characters were generated
	for _, expectedChar := range chars {
		if counts[byte(expectedChar)] == 0 {
			t.Errorf("Character '%c' was never generated", expectedChar)
		}
	}

	// Basic distribution check - each character should appear at least 10% of the time
	minExpected := iterations / 10
	for char, count := range counts {
		if count < minExpected {
			t.Errorf("Character '%c' appeared only %d times (expected at least %d)", char, count, minExpected)
		}
	}
}

func TestShuffleBytes(t *testing.T) {
	original := []byte("ABCD1234")
	data := make([]byte, len(original))
	copy(data, original)

	err := shuffleBytes(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that all original characters are still present
	originalStr := string(original)
	shuffledStr := string(data)

	for _, char := range originalStr {
		if !strings.ContainsRune(shuffledStr, char) {
			t.Errorf("Character '%c' missing after shuffle", char)
		}
	}

	// Length should be unchanged
	if len(data) != len(original) {
		t.Errorf("Length changed after shuffle: expected %d, got %d", len(original), len(data))
	}
}

func TestShuffleBytesRandomness(t *testing.T) {
	original := []byte("ABCDEFGH12345678")

	// Test multiple shuffles to check for randomness
	results := make(map[string]int)
	const iterations = 100

	for i := 0; i < iterations; i++ {
		data := make([]byte, len(original))
		copy(data, original)

		err := shuffleBytes(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		results[string(data)]++
	}

	// Should produce multiple different arrangements
	// (very unlikely to get same arrangement many times by chance)
	if len(results) < 10 {
		t.Errorf("Shuffle appears non-random: only %d unique arrangements in %d iterations", len(results), iterations)
	}
}

// Benchmark tests
func BenchmarkGenerateSecureString(b *testing.B) {
	lengths := []int{8, 16, 32, 64}

	for _, length := range lengths {
		b.Run(fmt.Sprintf("length_%d", length), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := GenerateSecureString(length)
				if err != nil {
					b.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func BenchmarkGetRandomChar(b *testing.B) {
	const chars = uppercaseLetters + lowercaseLetters + digits + specialChars

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := getRandomChar(chars)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
