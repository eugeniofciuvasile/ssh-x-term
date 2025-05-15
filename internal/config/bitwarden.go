package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type BitwardenConfig struct {
	ServerURL string
	Email     string
}

type BitwardenManager struct {
	cfg        *BitwardenConfig
	session    string
	authed     bool
	vaultMutex sync.Mutex // thread safety
	items      map[string]SSHConnection
}

const sshNoteTagField = "ssh-x-term"

func NewBitwardenManager(cfg *BitwardenConfig) (*BitwardenManager, error) {
	return &BitwardenManager{
		cfg:   cfg,
		items: make(map[string]SSHConnection),
	}, nil
}

// checkBwCLI checks if the 'bw' CLI is available in the PATH.
func checkBwCLI() error {
	_, err := exec.LookPath("bw")
	if err != nil {
		return errors.New("Bitwarden CLI (`bw`) is not installed or not in your PATH. Please install it: https://bitwarden.com/help/cli/")
	}
	return nil
}

func (bwm *BitwardenManager) Login(password, otp string) error {
	if err := checkBwCLI(); err != nil {
		return err
	}
	args := []string{"login", bwm.cfg.Email, password, "--raw"}
	if bwm.cfg.ServerURL != "" {
		args = append(args, "--server", bwm.cfg.ServerURL)
	}
	if otp != "" {
		args = append(args, "--code", otp)
	}
	cmd := exec.Command("bw", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Bitwarden login failed: %s", stderr.String())
	}
	bwm.session = strings.TrimSpace(out.String())
	bwm.authed = true
	return nil
}

func (bwm *BitwardenManager) Unlock(password string) error {
	if err := checkBwCLI(); err != nil {
		return err
	}
	cmd := exec.Command("bw", "unlock", password, "--raw")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Bitwarden unlock failed: %s", stderr.String())
	}
	bwm.session = strings.TrimSpace(out.String())
	bwm.authed = true
	return nil
}

func (bwm *BitwardenManager) SessionKey() (string, error) {
	if !bwm.authed || bwm.session == "" {
		return "", fmt.Errorf("Bitwarden session is not authenticated")
	}
	return bwm.session, nil
}

// ---- Storage Interface Implementation ----

// Load fetches all SSH connections from Bitwarden
func (bwm *BitwardenManager) Load() error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()

	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}

	cmd := exec.Command("bw", "list", "items", "--session", session)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bw list items failed: %s", stderr.String())
	}

	var allItems []map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &allItems); err != nil {
		return fmt.Errorf("failed to parse bw list items JSON: %w", err)
	}

	bwm.items = make(map[string]SSHConnection)
	for _, item := range allItems {
		// Only login items
		if t, ok := item["type"].(float64); !ok || int(t) != 1 {
			continue
		}
		notes, _ := item["notes"].(string)
		if notes == "" {
			continue
		}
		conn := SSHConnection{}
		if err := json.Unmarshal([]byte(notes), &conn); err != nil {
			// Not one of our items
			continue
		}
		if id, ok := item["id"].(string); ok {
			conn.ID = id
		}
		bwm.items[conn.ID] = conn
	}
	return nil
}

func (bwm *BitwardenManager) AddConnection(conn SSHConnection) error {
	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}
	connJSON, err := json.Marshal(conn)
	if err != nil {
		return err
	}
	login := map[string]interface{}{
		"username": conn.Username,
		"password": conn.Password,
		"uris": []map[string]interface{}{
			{
				"match": nil,
				"uri":   fmt.Sprintf("ssh://%s", conn.Host),
			},
		},
	}
	item := map[string]interface{}{
		"type":     1, // Login
		"name":     conn.Name,
		"notes":    string(connJSON),
		"favorite": false,
		"fields":   []map[string]interface{}{},
		"login":    login,
	}
	itemJSON, err := json.Marshal(item)
	if err != nil {
		return err
	}
	encodeCmd := exec.Command("bw", "encode")
	encodeCmd.Stdin = bytes.NewReader(itemJSON)
	var encodedOutput, encodeErr bytes.Buffer
	encodeCmd.Stdout = &encodedOutput
	encodeCmd.Stderr = &encodeErr
	if err := encodeCmd.Run(); err != nil {
		return fmt.Errorf("failed to encode Bitwarden item: %s - %s", err, encodeErr.String())
	}
	createCmd := exec.Command("bw", "create", "item", "--session", session)
	createCmd.Stdin = bytes.NewReader(encodedOutput.Bytes())
	var createOut, createErr bytes.Buffer
	createCmd.Stdout = &createOut
	createCmd.Stderr = &createErr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create Bitwarden item: %s - %s", err, createErr.String())
	}
	return bwm.Load()
}

// Save is a no-op for Bitwarden (all changes go through CLI immediately)
func (bwm *BitwardenManager) Save() error {
	return nil
}

// DeleteConnection removes a connection by Bitwarden item ID
func (bwm *BitwardenManager) DeleteConnection(id string) error {
	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}
	cmd := exec.Command("bw", "delete", "item", id, "--session", session, "--permanent")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not delete Bitwarden item: %s", stderr.String())
	}
	// Reload items from vault
	return bwm.Load()
}

// GetConnection returns a connection by ID
func (bwm *BitwardenManager) GetConnection(id string) (SSHConnection, bool) {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	c, ok := bwm.items[id]
	return c, ok
}

// ListConnections returns all connections
func (bwm *BitwardenManager) ListConnections() []SSHConnection {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	conns := make([]SSHConnection, 0, len(bwm.items))
	for _, c := range bwm.items {
		conns = append(conns, c)
	}
	return conns
}

// EditConnection updates an existing SSH connection (not part of Storage interface, but useful)
func (bwm *BitwardenManager) EditConnection(conn SSHConnection) error {
	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}
	if conn.ID == "" {
		return fmt.Errorf("missing Bitwarden item ID for edit")
	}
	connBytes, err := json.Marshal(conn)
	if err != nil {
		return err
	}
	// Fetch current item
	cmdGet := exec.Command("bw", "get", "item", conn.ID, "--session", session)
	var outGet, errGet bytes.Buffer
	cmdGet.Stdout = &outGet
	cmdGet.Stderr = &errGet
	if err := cmdGet.Run(); err != nil {
		return fmt.Errorf("could not fetch Bitwarden item: %s", errGet.String())
	}
	var item map[string]interface{}
	if err := json.Unmarshal(outGet.Bytes(), &item); err != nil {
		return err
	}
	item["notes"] = string(connBytes)
	item["name"] = conn.Name
	if tags, ok := item["tags"].([]interface{}); ok {
		found := false
		for _, tag := range tags {
			if tagstr, ok := tag.(string); ok && tagstr == sshNoteTagField {
				found = true
				break
			}
		}
		if !found {
			item["tags"] = append(tags, sshNoteTagField)
		}
	} else {
		item["tags"] = []string{sshNoteTagField}
	}
	itemBytes, err := json.Marshal(item)
	if err != nil {
		return err
	}
	cmdEdit := exec.Command("bw", "edit", "item", conn.ID, "--session", session)
	cmdEdit.Stdin = bytes.NewReader(itemBytes)
	var outEdit, errEdit bytes.Buffer
	cmdEdit.Stdout = &outEdit
	cmdEdit.Stderr = &errEdit
	if err := cmdEdit.Run(); err != nil {
		return fmt.Errorf("could not edit Bitwarden item: %s", errEdit.String())
	}
	// Reload items from vault
	return bwm.Load()
}

func (bwm *BitwardenManager) Status() (loggedIn bool, unlocked bool, err error) {
	if err := checkBwCLI(); err != nil {
		return false, false, err
	}
	cmd := exec.Command("bw", "status")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, false, fmt.Errorf("bw status failed: %s", stderr.String())
	}
	type bwStatus struct {
		Status string `json:"status"`
	}
	var stat bwStatus
	if err := json.Unmarshal(out.Bytes(), &stat); err != nil {
		return false, false, err
	}
	switch stat.Status {
	case "unauthenticated":
		return false, false, nil
	case "locked":
		return true, false, nil
	case "unlocked":
		return true, true, nil
	}
	return false, false, fmt.Errorf("unknown Bitwarden status: %s", stat.Status)
}
