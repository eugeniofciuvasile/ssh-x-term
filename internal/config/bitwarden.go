package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	vaultMutex sync.Mutex
	items      map[string]SSHConnection
}

func NewBitwardenManager(cfg *BitwardenConfig) (*BitwardenManager, error) {
	return &BitwardenManager{
		cfg:   cfg,
		items: make(map[string]SSHConnection),
	}, nil
}

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

	var allItems []map[string]any
	if err := json.Unmarshal(out.Bytes(), &allItems); err != nil {
		return fmt.Errorf("failed to parse bw list items JSON: %w", err)
	}

	bwm.items = make(map[string]SSHConnection)
	for _, item := range allItems {
		if t, ok := item["type"].(float64); !ok || int(t) != 1 {
			continue
		}
		conn := SSHConnection{}
		if id, ok := item["id"].(string); ok {
			conn.ID = id
		}
		if name, ok := item["name"].(string); ok {
			conn.Name = name
		}
		login, ok := item["login"].(map[string]any)
		if ok {
			if username, ok := login["username"].(string); ok {
				conn.Username = username
			}
			if password, ok := login["password"].(string); ok {
				conn.Password = password
			}
			if uris, ok := login["uris"].([]any); ok && len(uris) > 0 {
				if first, ok := uris[0].(map[string]any); ok {
					if uri, ok := first["uri"].(string); ok {
						rest := strings.TrimPrefix(uri, "ssh://")
						hostport := strings.Split(rest, ":")
						conn.Host = hostport[0]
						if len(hostport) > 1 {
							if port, err := strconv.Atoi(hostport[1]); err == nil {
								conn.Port = port
							}
						} else {
							conn.Port = 22
						}
					}
				}
			}
		}
		if fields, ok := item["fields"].([]any); ok {
			for _, f := range fields {
				if field, ok := f.(map[string]any); ok {
					name, _ := field["name"].(string)
					value, _ := field["value"].(string)
					if strings.ToLower(name) == "use_password" {
						conn.UsePassword = value == "true"
					}
				}
			}
		}
		if notes, ok := item["notes"].(string); ok && notes != "" {
			conn.PublicKey = notes
		}
		bwm.items[conn.ID] = conn
	}
	return nil
}

func (bwm *BitwardenManager) Save() error {
	return nil
}

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
	return bwm.Load()
}

func (bwm *BitwardenManager) GetConnection(id string) (SSHConnection, bool) {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	c, ok := bwm.items[id]
	return c, ok
}

func (bwm *BitwardenManager) ListConnections() []SSHConnection {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	conns := make([]SSHConnection, 0, len(bwm.items))
	for _, c := range bwm.items {
		conns = append(conns, c)
	}
	return conns
}

func (bwm *BitwardenManager) AddConnection(conn SSHConnection) error {
	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}

	publicKey := conn.PublicKey
	privateKey := conn.Password

	if !conn.UsePassword {
		if privateKey == "" && conn.KeyFile != "" {
			expanded := ExpandPath(conn.KeyFile)
			keyData, err := os.ReadFile(expanded)
			if err != nil {
				tip := ""
				if os.IsPermission(err) {
					tip = " (permission denied: check file permissions, you may need to run as the user who owns the file or change ownership)"
				}
				return fmt.Errorf("could not read private key file '%s': %v%s", conn.KeyFile, err, tip)
			}
			privateKey = string(keyData)
		}
		pubPath := ExpandPath(conn.KeyFile) + ".pub"
		if publicKey == "" {
			pubData, err := os.ReadFile(pubPath)
			if err != nil {
				if !os.IsNotExist(err) {
					tip := ""
					if os.IsPermission(err) {
						tip = " (permission denied: check file permissions, you may need to run as the user who owns the file or change ownership)"
					}
					return fmt.Errorf("could not read public key file '%s': %v%s", pubPath, err, tip)
				}
			} else {
				publicKey = string(pubData)
			}
		}
	}

	fields := []map[string]any{
		{
			"name":  "use_password",
			"value": strconv.FormatBool(conn.UsePassword),
			"type":  0,
		},
	}

	login := map[string]any{
		"username": conn.Username,
		"password": privateKey,
		"uris": []map[string]any{
			{
				"uri": fmt.Sprintf("ssh://%s:%d", conn.Host, conn.Port),
			},
		},
	}

	item := map[string]any{
		"type":   1, // Login
		"name":   conn.Name,
		"login":  login,
		"fields": fields,
		"notes":  publicKey,
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

func (bwm *BitwardenManager) EditConnection(conn SSHConnection) error {
	session, err := bwm.SessionKey()
	if err != nil {
		return err
	}
	if conn.ID == "" {
		return fmt.Errorf("missing Bitwarden item ID for edit")
	}

	publicKey := conn.PublicKey
	privateKey := conn.Password

	if !conn.UsePassword {
		if privateKey == "" && conn.KeyFile != "" {
			expanded := ExpandPath(conn.KeyFile)
			keyData, err := os.ReadFile(expanded)
			if err != nil {
				tip := ""
				if os.IsPermission(err) {
					tip = " (permission denied: check file permissions, you may need to run as the user who owns the file or change ownership)"
				}
				return fmt.Errorf("could not read private key file '%s': %v%s", conn.KeyFile, err, tip)
			}
			privateKey = string(keyData)
		}
		pubPath := ExpandPath(conn.KeyFile) + ".pub"
		if publicKey == "" {
			pubData, err := os.ReadFile(pubPath)
			if err != nil {
				if !os.IsNotExist(err) {
					tip := ""
					if os.IsPermission(err) {
						tip = " (permission denied: check file permissions, you may need to run as the user who owns the file or change ownership)"
					}
					return fmt.Errorf("could not read public key file '%s': %v%s", pubPath, err, tip)
				}
			} else {
				publicKey = string(pubData)
			}
		}
	}

	fields := []map[string]any{
		{
			"name":  "use_password",
			"value": strconv.FormatBool(conn.UsePassword),
			"type":  0,
		},
	}

	login := map[string]any{
		"username": conn.Username,
		"password": privateKey,
		"uris": []map[string]any{
			{
				"uri": fmt.Sprintf("ssh://%s:%d", conn.Host, conn.Port),
			},
		},
	}

	item := map[string]any{
		"type":   1, // Login
		"name":   conn.Name,
		"login":  login,
		"fields": fields,
		"notes":  publicKey,
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
		errMsg := fmt.Sprintf("failed to encode Bitwarden item: %s - %s", err, encodeErr.String())
		return fmt.Errorf(errMsg)
	}

	editCmd := exec.Command("bw", "edit", "item", conn.ID, "--session", session)
	editCmd.Stdin = bytes.NewReader(encodedOutput.Bytes())
	var editOut, editErr bytes.Buffer
	editCmd.Stdout = &editOut
	editCmd.Stderr = &editErr

	if err := editCmd.Run(); err != nil {
		errMsg := fmt.Sprintf("could not edit Bitwarden item: %s - %s", err, editErr.String())
		return fmt.Errorf(errMsg)
	}

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
