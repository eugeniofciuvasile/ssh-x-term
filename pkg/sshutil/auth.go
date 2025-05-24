package sshutil

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// GetKeyAuthMethod returns an AuthMethod using the specified private key file
func GetKeyAuthMethod(keyPath string) (ssh.AuthMethod, error) {
	// If keyPath is empty, try to use default key location
	if keyPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Failed to get user home directory: %v", err)
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		keyPath = filepath.Join(homeDir, ".ssh", "id_rsa")
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		log.Printf("Failed to read private key from %s: %v", keyPath, err)
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		log.Printf("Failed to parse private key %s: %v", keyPath, err)
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// GetPasswordAuthMethod returns an AuthMethod using the specified password
func GetPasswordAuthMethod(password string) ssh.AuthMethod {
	// Password is not logged.
	return ssh.Password(password)
}

// GetAuthMethod returns an appropriate AuthMethod based on the provided options
func GetAuthMethod(usePassword bool, password, keyFile string) (ssh.AuthMethod, error) {
	if usePassword {
		if password == "" {
			log.Printf("Password is required when usePassword is true")
			return nil, errors.New("password is required when usePassword is true")
		}
		return GetPasswordAuthMethod(password), nil
	}

	return GetKeyAuthMethod(keyFile)
}
