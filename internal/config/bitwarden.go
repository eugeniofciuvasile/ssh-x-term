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

// https://api.bitwarden.com default login URL
func (bwm *BitwardenManager) Login(password, otp string) error {
	if err := checkBwCLI(); err != nil {
		fmt.Println(err)
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
		fmt.Printf("Bitwarden login failed: %s\n", stderr.String())
		return fmt.Errorf("Bitwarden login failed: %s", stderr.String())
	}
	bwm.session = strings.TrimSpace(out.String())
	bwm.authed = true
	return nil
}

func (bwm *BitwardenManager) Unlock(password string) error {
	if err := checkBwCLI(); err != nil {
		fmt.Println(err)
		return err
	}
	cmd := exec.Command("bw", "unlock", password, "--raw")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Bitwarden unlock failed: %s\n", stderr.String())
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

	cmd := exec.Command("bw", "list", "items", "--search", sshNoteTagField, "--session", session)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bw list items failed: %s", stderr.String())
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &items); err != nil {
		return err
	}

	bwm.items = make(map[string]SSHConnection)
	for _, item := range items {
		if notes, ok := item["notes"].(string); ok && notes != "" {
			conn := SSHConnection{}
			if err := json.Unmarshal([]byte(notes), &conn); err == nil {
				if id, ok := item["id"].(string); ok {
					conn.ID = id
				}
				bwm.items[conn.ID] = conn
			}
		}
	}
	return nil
}

// Save is a no-op for Bitwarden (all changes go through CLI immediately)
func (bwm *BitwardenManager) Save() error {
	return nil
}

// AddConnection creates a new SSH connection
func (bwm *BitwardenManager) AddConnection(conn SSHConnection) error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()

	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}
	connBytes, err := json.Marshal(conn)
	if err != nil {
		return err
	}
	item := map[string]interface{}{
		"type":  2, // Secure Note
		"name":  conn.Name,
		"notes": string(connBytes),
		"tags":  []string{sshNoteTagField},
	}
	itemBytes, err := json.Marshal(item)
	if err != nil {
		return err
	}
	cmd := exec.Command("bw", "create", "item", "--session", session)
	cmd.Stdin = bytes.NewReader(itemBytes)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Bitwarden item: %s", stderr.String())
	}
	// Optionally reload items to update the internal map
	_ = bwm.Load()
	return nil
}

// DeleteConnection removes a connection by Bitwarden item ID
func (bwm *BitwardenManager) DeleteConnection(id string) error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()

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
	delete(bwm.items, id)
	return nil
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
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()

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
			if tag.(string) == sshNoteTagField {
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
	// Optionally reload items
	_ = bwm.Load()
	return nil
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
