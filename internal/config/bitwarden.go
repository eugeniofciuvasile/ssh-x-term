package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
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
	cfg                *BitwardenConfig
	session            string
	authed             bool
	vaultMutex         sync.Mutex
	items              map[string]SSHConnection
	organizations      []Organization
	collections        []Collection
	personalVault      bool
	selectedCollection *Collection
}

func NewBitwardenManager(cfg *BitwardenConfig) (*BitwardenManager, error) {
	return &BitwardenManager{
		cfg:                cfg,
		items:              make(map[string]SSHConnection),
		selectedCollection: nil,
		personalVault:      false,
	}, nil
}

func checkBwCLI() error {
	_, err := exec.LookPath("bw")
	if err != nil {
		log.Print("Bitwarden CLI (`bw`) is not installed or not in your PATH. Please install it: https://bitwarden.com/help/cli/")
		return errors.New("Bitwarden CLI (`bw`) is not installed or not in your PATH. Please install it: https://bitwarden.com/help/cli/")
	}
	return nil
}

func (bwm *BitwardenManager) Login(password, otp string) error {
	if err := checkBwCLI(); err != nil {
		log.Print("Bitwarden CLI check failed during login")
		return err
	}
	if bwm.cfg.ServerURL != "" {
		cmd := exec.Command("bw", "config", "server", bwm.cfg.ServerURL)
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Bitwarden config server failed: %s", stderr.String())
			return errors.New("Bitwarden config server failed: " + stderr.String())
		}
	}

	args := []string{"login", bwm.cfg.Email, password, "--raw"}
	if otp != "" {
		args = append(args, "--code", otp)
	}
	cmd := exec.Command("bw", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("Bitwarden login failed: %s", stderr.String())
		return errors.New("Bitwarden login failed: " + stderr.String())
	}
	bwm.session = strings.TrimSpace(out.String())
	bwm.authed = true
	return nil
}

func (bwm *BitwardenManager) Unlock(password string) error {
	if err := checkBwCLI(); err != nil {
		log.Print("Bitwarden CLI check failed during unlock")
		return err
	}
	cmd := exec.Command("bw", "unlock", password, "--raw")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("Bitwarden unlock failed: %s", stderr.String())
		return errors.New("Bitwarden unlock failed: " + stderr.String())
	}
	bwm.session = strings.TrimSpace(out.String())
	bwm.authed = true
	return nil
}

func (bwm *BitwardenManager) SessionKey() (string, error) {
	if !bwm.authed || bwm.session == "" {
		log.Print("Tried to fetch Bitwarden session key, but not authenticated")
		return "", errors.New("Bitwarden session is not authenticated")
	}
	return bwm.session, nil
}

func (bmw *BitwardenManager) SetPersonalVault(value bool) {
	bmw.personalVault = value
}

func (bwm *BitwardenManager) IsPersonalVault() bool {
	return bwm.personalVault
}

func (bwm *BitwardenManager) GetSelectedCollection() *Collection {
	return bwm.selectedCollection
}

func (bwm *BitwardenManager) SetSelectedCollection(collection *Collection) {
	bwm.selectedCollection = collection
}

// ---- Storage Interface Implementation ----

func (bwm *BitwardenManager) Load() error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()

	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during Load")
		return err
	}

	cmd := exec.Command("bw", "list", "items", "--session", session, "--organizationid", "null")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("bw list items failed: %s", stderr.String())
		return errors.New("bw list items failed: " + stderr.String())
	}

	var allItems []map[string]any
	if err := json.Unmarshal(out.Bytes(), &allItems); err != nil {
		log.Printf("Failed to parse bw list items JSON: %v", err)
		return err
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
					if uri, ok := first["uri"].(string); ok && strings.HasPrefix(uri, "ssh://") {
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
					} else {
						continue
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
		log.Print("Could not get Bitwarden session key during DeleteConnection")
		return err
	}
	cmd := exec.Command("bw", "delete", "item", id, "--session", session, "--permanent")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Could not delete Bitwarden item: %s", stderr.String())
		return errors.New("could not delete Bitwarden item: " + stderr.String())
	}
	return bwm.Load()
}

func (bwm *BitwardenManager) GetConnection(id string) (SSHConnection, bool) {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	c, ok := bwm.items[id]
	return c, ok
}

func (bwm *BitwardenManager) AddConnection(conn SSHConnection) error {
	return bwm.AddConnectionInCollectionAndOrganization(conn, "", "")
}

func (bwm *BitwardenManager) AddConnectionInCollectionAndOrganization(conn SSHConnection, organizationID, collectionID string) error {
	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during AddConnectionInCollectionAndOrganization")
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
				log.Printf("Could not read private key file '%s': %v%s", conn.KeyFile, err, tip)
				return errors.New("could not read private key file '" + conn.KeyFile + "': " + err.Error() + tip)
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
					log.Printf("Could not read public key file '%s': %v%s", pubPath, err, tip)
					return errors.New("could not read public key file '" + pubPath + "': " + err.Error() + tip)
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
				"uri": "ssh://" + conn.Host + ":" + strconv.Itoa(conn.Port),
			},
		},
	}

	var item map[string]any
	if collectionID != "" && organizationID != "" {
		item = map[string]any{
			"type":           1, // Login
			"name":           conn.Name,
			"login":          login,
			"fields":         fields,
			"notes":          publicKey,
			"collectionIds":  []string{collectionID},
			"organizationId": organizationID,
		}
	} else {
		item = map[string]any{
			"type":   1, // Login
			"name":   conn.Name,
			"login":  login,
			"fields": fields,
			"notes":  publicKey,
		}
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		log.Printf("JSON marshaling failed for Bitwarden item: %v", err)
		return err
	}

	encodeCmd := exec.Command("bw", "encode")
	encodeCmd.Stdin = bytes.NewReader(itemJSON)
	var encodedOutput, encodeErr bytes.Buffer
	encodeCmd.Stdout = &encodedOutput
	encodeCmd.Stderr = &encodeErr
	if err := encodeCmd.Run(); err != nil {
		log.Printf("Failed to encode Bitwarden item: %s - %s", err, encodeErr.String())
		return errors.New("failed to encode Bitwarden item: " + err.Error() + " - " + encodeErr.String())
	}

	createCmd := exec.Command("bw", "create", "item", "--session", session)
	createCmd.Stdin = bytes.NewReader(encodedOutput.Bytes())
	var createOut, createErr bytes.Buffer
	createCmd.Stdout = &createOut
	createCmd.Stderr = &createErr
	if err := createCmd.Run(); err != nil {
		log.Printf("Failed to create Bitwarden item: %s - %s", err, createErr.String())
		return errors.New("failed to create Bitwarden item: " + err.Error() + " - " + createErr.String())
	}
	if bwm.IsPersonalVault() {
		return bwm.Load()
	} else {
		return bwm.LoadConnectionsByCollectionId(collectionID)
	}
}

func (bwm *BitwardenManager) EditConnection(conn SSHConnection) error {
	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during EditConnection")
		return err
	}
	if conn.ID == "" {
		log.Print("Missing Bitwarden item ID for edit")
		return errors.New("missing Bitwarden item ID for edit")
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
				log.Printf("Could not read private key file '%s': %v%s", conn.KeyFile, err, tip)
				return errors.New("could not read private key file '" + conn.KeyFile + "': " + err.Error() + tip)
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
					log.Printf("Could not read public key file '%s': %v%s", pubPath, err, tip)
					return errors.New("could not read public key file '" + pubPath + "': " + err.Error() + tip)
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
				"uri": "ssh://" + conn.Host + ":" + strconv.Itoa(conn.Port),
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
		log.Printf("JSON marshaling failed for Bitwarden item: %v", err)
		return err
	}

	encodeCmd := exec.Command("bw", "encode")
	encodeCmd.Stdin = bytes.NewReader(itemJSON)
	var encodedOutput, encodeErr bytes.Buffer
	encodeCmd.Stdout = &encodedOutput
	encodeCmd.Stderr = &encodeErr

	if err := encodeCmd.Run(); err != nil {
		log.Printf("Failed to encode Bitwarden item: %s - %s", err, encodeErr.String())
		return errors.New("failed to encode Bitwarden item: " + err.Error() + " - " + encodeErr.String())
	}

	editCmd := exec.Command("bw", "edit", "item", conn.ID, "--session", session)
	editCmd.Stdin = bytes.NewReader(encodedOutput.Bytes())
	var editOut, editErr bytes.Buffer
	editCmd.Stdout = &editOut
	editCmd.Stderr = &editErr

	if err := editCmd.Run(); err != nil {
		log.Printf("Could not edit Bitwarden item: %s - %s", err, editErr.String())
		return errors.New("could not edit Bitwarden item: " + err.Error() + " - " + editErr.String())
	}

	if bwm.IsPersonalVault() {
		return bwm.Load()
	} else {
		var collectionID string
		if bwm.GetSelectedCollection() != nil {
			collectionID = bwm.GetSelectedCollection().ID
		}
		return bwm.LoadConnectionsByCollectionId(collectionID)
	}
}

func (bwm *BitwardenManager) Status() (loggedIn bool, unlocked bool, err error) {
	if err := checkBwCLI(); err != nil {
		log.Print("Bitwarden CLI check failed during Status")
		return false, false, err
	}
	cmd := exec.Command("bw", "status")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("bw status failed: %s", stderr.String())
		return false, false, errors.New("bw status failed: " + stderr.String())
	}
	type bwStatus struct {
		Status string `json:"status"`
	}
	var stat bwStatus
	if err := json.Unmarshal(out.Bytes(), &stat); err != nil {
		log.Printf("Failed to parse Bitwarden status JSON: %v", err)
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
	log.Printf("Unknown Bitwarden status: %s", stat.Status)
	return false, false, errors.New("unknown Bitwarden status: " + stat.Status)
}

func (bwm *BitwardenManager) Sync() error {
	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during Sync")
		return err
	}
	cmd := exec.Command("bw", "sync", "--session", session)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Could not sync Bitwarden vault: %s", stderr.String())
		return errors.New("could not sync Bitwarden vault: " + stderr.String())
	}
	return nil
}

func (bwm *BitwardenManager) LoadOrganizations() error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	if err := bwm.Sync(); err != nil {
		return err
	}
	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during LoadOrganizations")
		return err
	}
	cmd := exec.Command("bw", "list", "organizations", "--session", session)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Could not list organizations: %s", stderr.String())
		return errors.New("could not list organizations: " + stderr.String())
	}
	var orgs []Organization
	if err := json.Unmarshal(out.Bytes(), &orgs); err != nil {
		log.Printf("Failed to parse organizations JSON: %v", err)
		return err
	}
	bwm.organizations = orgs
	return nil
}

func (bwm *BitwardenManager) ListOrganizations() []Organization {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	return bwm.organizations
}

func (bwm *BitwardenManager) LoadCollectionsByOrganizationId(organizationId string) error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	if err := bwm.Sync(); err != nil {
		return err
	}
	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during LoadCollectionsByOrganizationId")
		return err
	}
	cmd := exec.Command("bw", "list", "collections", "--organizationid", organizationId, "--session", session)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Could not list collections: %s", stderr.String())
		return errors.New("could not list collections: " + stderr.String())
	}
	var collections []Collection
	if err := json.Unmarshal(out.Bytes(), &collections); err != nil {
		log.Printf("Failed to parse collections JSON: %v", err)
		return err
	}
	bwm.collections = collections
	return nil
}

func (bwm *BitwardenManager) ListCollections() []Collection {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	return bwm.collections
}

func (bwm *BitwardenManager) LoadConnectionsByCollectionId(collectionId string) error {
	bwm.vaultMutex.Lock()
	defer bwm.vaultMutex.Unlock()
	if err := bwm.Sync(); err != nil {
		return err
	}
	session, err := bwm.SessionKey()
	if err != nil {
		log.Print("Could not get Bitwarden session key during LoadConnectionsByCollectionId")
		return err
	}

	if collectionId == "" {
		collectionId = "null"
	}

	cmd := exec.Command("bw", "list", "items", "--collectionid", collectionId, "--session", session)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("bw list items failed: %s", stderr.String())
		return errors.New("bw list items failed: " + stderr.String())
	}

	var allItems []map[string]any
	if err := json.Unmarshal(out.Bytes(), &allItems); err != nil {
		log.Printf("Failed to parse bw list items JSON: %v", err)
		return err
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
		if collectionIds, ok := item["collectionIds"].([]any); ok {
			conn.CollectionIds = []string{}
			for _, cid := range collectionIds {
				if cidStr, ok := cid.(string); ok {
					conn.CollectionIds = append(conn.CollectionIds, cidStr)
				}
			}
		}
		if orgId, ok := item["organizationId"].(string); ok {
			conn.OrganizationID = orgId
		}
		bwm.items[conn.ID] = conn
	}
	return nil
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
