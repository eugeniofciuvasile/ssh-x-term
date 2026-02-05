package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHConfigParsing(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	// Create a test SSH config file
	configPath := filepath.Join(sshDir, "config")
	configContent := `#sxt:id=test-id-1
#sxt:name=Test Server 1
#sxt:notes=Test notes
#sxt:use_password=true
Host testserver1
    HostName 192.168.1.100
    Port 2222
    User testuser

#sxt:id=test-id-2
#sxt:name=Test Server 2
#sxt:use_password=false
Host testserver2
    HostName example.com
    User admin
    IdentityFile ~/.ssh/id_rsa

# Regular SSH config entry (not managed by sxt)
Host regularhost
    HostName regular.example.com
    User regularuser
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create SSH config manager
	scm, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create SSH config manager: %v", err)
	}

	// Load config
	if err := scm.Load(); err != nil {
		t.Fatalf("Failed to load SSH config: %v", err)
	}

	// Verify connections
	connections := scm.ListConnections()
	if len(connections) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(connections))
	}

	// Check first connection
	var conn1 *SSHConnection
	for _, conn := range connections {
		if conn.ID == "test-id-1" {
			conn1 = &conn
			break
		}
	}

	if conn1 == nil {
		t.Fatal("Connection with ID test-id-1 not found")
	}

	if conn1.Name != "Test Server 1" {
		t.Errorf("Expected name 'Test Server 1', got '%s'", conn1.Name)
	}
	if conn1.Host != "192.168.1.100" {
		t.Errorf("Expected host '192.168.1.100', got '%s'", conn1.Host)
	}
	if conn1.Port != 2222 {
		t.Errorf("Expected port 2222, got %d", conn1.Port)
	}
	if conn1.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", conn1.Username)
	}
	if !conn1.UsePassword {
		t.Error("Expected UsePassword to be true")
	}
	if conn1.Notes != "Test notes" {
		t.Errorf("Expected notes 'Test notes', got '%s'", conn1.Notes)
	}

	// Check second connection
	var conn2 *SSHConnection
	for _, conn := range connections {
		if conn.ID == "test-id-2" {
			conn2 = &conn
			break
		}
	}

	if conn2 == nil {
		t.Fatal("Connection with ID test-id-2 not found")
	}

	if conn2.UsePassword {
		t.Error("Expected UsePassword to be false")
	}
	if conn2.KeyFile != "~/.ssh/id_rsa" {
		t.Errorf("Expected KeyFile '~/.ssh/id_rsa', got '%s'", conn2.KeyFile)
	}
}

func TestSSHConfigWriting(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create SSH config manager
	scm, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create SSH config manager: %v", err)
	}

	// Add a connection
	conn := SSHConnection{
		ID:          "test-write-1",
		Name:        "Write Test",
		Host:        "writetest.example.com",
		Port:        22,
		Username:    "writeuser",
		UsePassword: true,
		Notes:       "Write test notes",
	}

	if err := scm.AddConnection(conn); err != nil {
		t.Fatalf("Failed to add connection: %v", err)
	}

	// Load and verify
	scm2, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create second SSH config manager: %v", err)
	}

	if err := scm2.Load(); err != nil {
		t.Fatalf("Failed to load SSH config: %v", err)
	}

	connections := scm2.ListConnections()
	if len(connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(connections))
	}

	if connections[0].Name != "Write Test" {
		t.Errorf("Expected name 'Write Test', got '%s'", connections[0].Name)
	}
}

func TestSSHConfigPreservesNonManagedEntries(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	// Create a test SSH config with mixed entries
	configPath := filepath.Join(sshDir, "config")
	configContent := `# Regular SSH config entry
Host regular1
    HostName regular.example.com
    User regularuser

#sxt:id=managed-1
#sxt:name=Managed Entry
#sxt:use_password=false
Host managed1
    HostName managed.example.com
    User manageduser
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Load and save to trigger rewrite
	scm, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create SSH config manager: %v", err)
	}

	if err := scm.Load(); err != nil {
		t.Fatalf("Failed to load SSH config: %v", err)
	}

	if err := scm.Save(); err != nil {
		t.Fatalf("Failed to save SSH config: %v", err)
	}

	// Read the file and verify all entries have sxt metadata now
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)

	// Count Host entries
	hostCount := 0
	sxtIDCount := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Host ") {
			hostCount++
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#sxt:id=") {
			sxtIDCount++
		}
	}

	if hostCount != 2 {
		t.Errorf("Expected 2 Host entries, got %d", hostCount)
	}

	if sxtIDCount != 2 {
		t.Errorf("Expected 2 entries with sxt:id, got %d. All entries should have metadata after save.", sxtIDCount)
	}

	// Verify backup was created
	backupFiles, err := filepath.Glob(configPath + ".backup.*")
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}
	if len(backupFiles) == 0 {
		t.Error("Expected backup file to be created")
	}
}

func TestSSHConfigFirstTimeLoadExistingEntries(t *testing.T) {
	// Simulate a user's existing SSH config without any sxt metadata
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	configPath := filepath.Join(sshDir, "config")
	configContent := `Host server1
    HostName 192.168.1.100
    Port 2222
    User admin

Host server2
    HostName example.com
    User testuser
    IdentityFile ~/.ssh/id_rsa

Host github.com
    User git
    IdentityFile ~/.ssh/github_key
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// First load
	scm, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create SSH config manager: %v", err)
	}

	if err := scm.Load(); err != nil {
		t.Fatalf("Failed to load SSH config: %v", err)
	}

	connections := scm.ListConnections()
	if len(connections) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(connections))
	}

	// First save - should create metadata for all entries
	if err := scm.Save(); err != nil {
		t.Fatalf("Failed to save SSH config: %v", err)
	}

	// Verify backup was created
	backupFiles1, err := filepath.Glob(configPath + ".backup.*")
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}
	if len(backupFiles1) != 1 {
		t.Errorf("Expected 1 backup file after first save, got %d", len(backupFiles1))
	}

	// Verify migration marker was created
	markerPath := filepath.Join(tmpDir, ".config", "ssh-x-term", ".migration_done")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("Migration marker file was not created")
	}

	// Second load - verify no duplicates
	scm2, err := NewSSHConfigManager()
	if err != nil {
		t.Fatalf("Failed to create second SSH config manager: %v", err)
	}

	if err := scm2.Load(); err != nil {
		t.Fatalf("Failed to load SSH config second time: %v", err)
	}

	connections2 := scm2.ListConnections()
	if len(connections2) != 3 {
		t.Errorf("Expected 3 connections after reload, got %d (duplicates detected!)", len(connections2))
	}

	// Second save - should NOT create another backup (migration already done)
	if err := scm2.Save(); err != nil {
		t.Fatalf("Failed to save SSH config second time: %v", err)
	}

	backupFiles2, err := filepath.Glob(configPath + ".backup.*")
	if err != nil {
		t.Fatalf("Failed to glob backup files after second save: %v", err)
	}
	if len(backupFiles2) != 1 {
		t.Errorf("Expected still 1 backup file after second save, got %d (should not create more backups after migration)", len(backupFiles2))
	}

	// Verify all have IDs
	for _, conn := range connections2 {
		if conn.ID == "" {
			t.Errorf("Connection %s has no ID", conn.Name)
		}
	}
}
