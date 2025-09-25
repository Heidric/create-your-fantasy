package security

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	uppercaseLetters = "ACDEFGHJKMPQRTWXYZ"
	lowercaseLetters = "acdefghjkpqrtwxyz"
	digits           = "23479"
	specialChars     = "!@#%^&*_+-={}.,<>?"
)

// getRandomChar returns a random element from the given string
func getRandomChar(chars string) (byte, error) {
	if len(chars) == 0 {
		return 0, fmt.Errorf("source string is empty")
	}

	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
	if err != nil {
		return 0, err
	}

	return chars[index.Int64()], nil
}

// shuffleBytes shuffles the slice of bytes using the Fisher–Yates shuffle
func shuffleBytes(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		data[i], data[j.Int64()] = data[j.Int64()], data[i]
	}
	return nil
}

// GenerateSecureString generates a string of given length guaranteeing
// at least one element of each collection of symbols is present
func GenerateSecureString(length int) (string, error) {
	if length < 4 {
		return "", fmt.Errorf("length has to be at least 4")
	}

	allChars := uppercaseLetters + lowercaseLetters + digits + specialChars
	result := make([]byte, length)
	categories := []string{uppercaseLetters, lowercaseLetters, digits, specialChars}

	for i, category := range categories {
		char, err := getRandomChar(category)
		if err != nil {
			return "", err
		}
		result[i] = char
	}
	for i := 4; i < length; i++ {
		char, err := getRandomChar(allChars)
		if err != nil {
			return "", err
		}
		result[i] = char
	}
	if err := shuffleBytes(result); err != nil {
		return "", err
	}

	return string(result), nil
}
