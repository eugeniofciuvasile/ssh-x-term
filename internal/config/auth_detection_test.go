package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSSHConfigAuthDetection(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	configPath := filepath.Join(sshDir, "config")
	configContent := `# Password authentication (no IdentityFile)
Host password-server
    HostName 10.10.8.25
    User admin
    Port 22

# Key authentication (has IdentityFile)
Host key-server
    HostName example.com
    User keyuser
    IdentityFile ~/.ssh/id_rsa

# Another password server (no keys)
Host another-password
    HostName 192.168.1.100
    User testuser

# Key with PubkeyAuthentication
Host key-with-option
    HostName secure.example.com
    User secureuser
    PubkeyAuthentication yes
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	scm, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create SSH config manager: %v", err)
	}

	if err := scm.parseSSHConfig(); err != nil {
		t.Fatalf("Failed to parse SSH config: %v", err)
	}

	connections := scm.Config.Connections
	if len(connections) != 4 {
		t.Fatalf("Expected 4 connections, got %d", len(connections))
	}

	// Test password-server (no IdentityFile)
	var passwordServer *SSHConnection
	for i := range connections {
		if connections[i].HostPattern == "password-server" {
			passwordServer = &connections[i]
			break
		}
	}
	if passwordServer == nil {
		t.Fatal("password-server not found")
	}
	if !passwordServer.UsePassword {
		t.Error("password-server should have UsePassword=true (no IdentityFile)")
	}
	if passwordServer.KeyFile != "" {
		t.Errorf("password-server should have empty KeyFile, got: %s", passwordServer.KeyFile)
	}

	// Test key-server (has IdentityFile)
	var keyServer *SSHConnection
	for i := range connections {
		if connections[i].HostPattern == "key-server" {
			keyServer = &connections[i]
			break
		}
	}
	if keyServer == nil {
		t.Fatal("key-server not found")
	}
	if keyServer.UsePassword {
		t.Error("key-server should have UsePassword=false (has IdentityFile)")
	}
	if keyServer.KeyFile == "" {
		t.Error("key-server should have KeyFile set")
	}

	// Test another-password (no IdentityFile)
	var anotherPassword *SSHConnection
	for i := range connections {
		if connections[i].HostPattern == "another-password" {
			anotherPassword = &connections[i]
			break
		}
	}
	if anotherPassword == nil {
		t.Fatal("another-password not found")
	}
	if !anotherPassword.UsePassword {
		t.Error("another-password should have UsePassword=true (no IdentityFile)")
	}

	// Test key-with-option (has PubkeyAuthentication yes)
	var keyWithOption *SSHConnection
	for i := range connections {
		if connections[i].HostPattern == "key-with-option" {
			keyWithOption = &connections[i]
			break
		}
	}
	if keyWithOption == nil {
		t.Fatal("key-with-option not found")
	}
	if keyWithOption.UsePassword {
		t.Error("key-with-option should have UsePassword=false (PubkeyAuthentication yes)")
	}
}
