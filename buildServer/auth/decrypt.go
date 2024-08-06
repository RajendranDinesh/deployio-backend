package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

func Decrypt(cipherText string) (string, error) {
	key, keyExists := os.LookupEnv("ENV_SECRET")
	if !keyExists {
		return "", fmt.Errorf("[ENC] env secret is not accessible")
	}

	// Convert key and cipherText to bytes
	keyBytes := []byte(key)
	cipherBytes, err := hex.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	// Create a new AES cipher block
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore, it's common to include it at the beginning of the cipher text.
	if len(cipherBytes) < aes.BlockSize {
		return "", errors.New("cipherText too short")
	}
	iv := cipherBytes[:aes.BlockSize]
	cipherBytes = cipherBytes[aes.BlockSize:]

	// Decrypt the cipher text using CFB mode
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherBytes, cipherBytes)

	return string(cipherBytes), nil
}
