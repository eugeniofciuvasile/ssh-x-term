package ssh

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/pkg/sftp"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/ssh"
)

// FileInfo represents a file or directory with extended metadata
type FileInfo struct {
	Name    string
	Size    int64
	IsDir   bool
	Mode    string
	ModTime time.Time // Modification time
	Perm    string    // Permission string (e.g. drwxr-xr-x)
	Owner   string    // UID or Owner Name
	Group   string    // GID or Group Name
}

// SFTPClient wraps an SFTP client connection
type SFTPClient struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewSFTPClient creates a new SFTP client connection
func NewSFTPClient(connConfig config.SSHConnection) (*SFTPClient, error) {
	// If password-based authentication is enabled, retrieve the password from the keyring
	if connConfig.UsePassword && connConfig.Password == "" {
		password, err := keyring.Get(keyringService, connConfig.ID)
		if err != nil {
			log.Printf("Failed to retrieve password from keyring for connection ID %s: %v", connConfig.ID, err)
			return nil, fmt.Errorf("failed to retrieve password: %w", err)
		}
		connConfig.Password = password
	}

	// Create SSH client first
	client, err := NewClient(connConfig)
	if err != nil {
		// Check if it's a passphrase error - don't wrap it
		var passphraseErr *PassphraseRequiredError
		if errors.As(err, &passphraseErr) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}

	// Create SFTP client from SSH connection
	sftpClient, err := sftp.NewClient(client.conn)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return &SFTPClient{
		sshClient:  client.conn,
		sftpClient: sftpClient,
	}, nil
}

// GetWorkingDir returns the current working directory of the SFTP connection
func (s *SFTPClient) GetWorkingDir() (string, error) {
	if s.sftpClient == nil {
		return "", fmt.Errorf("SFTP client not connected")
	}
	return s.sftpClient.Getwd()
}

// Close closes the SFTP and SSH connections
func (s *SFTPClient) Close() error {
	var err error
	if s.sftpClient != nil {
		err = s.sftpClient.Close()
	}
	if s.sshClient != nil {
		if closeErr := s.sshClient.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// ListFiles lists files in a directory (Remote)
func (s *SFTPClient) ListFiles(path string) ([]FileInfo, error) {
	if s.sftpClient == nil {
		return nil, fmt.Errorf("SFTP client not connected")
	}

	entries, err := s.sftpClient.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		owner, group := getOwnerGroup(entry.Sys())

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    entry.Size(),
			IsDir:   entry.IsDir(),
			Mode:    entry.Mode().String(),
			ModTime: entry.ModTime(),
			Perm:    entry.Mode().String(),
			Owner:   owner,
			Group:   group,
		})
	}

	sortFiles(files)
	return files, nil
}

// ListLocalFiles lists files in a local directory
func ListLocalFiles(path string) ([]FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		owner, group := getOwnerGroup(info.Sys())

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime(),
			Perm:    info.Mode().String(),
			Owner:   owner,
			Group:   group,
		})
	}

	sortFiles(files)
	return files, nil
}

// sortFiles sorts directories first, then files, alphabetically
func sortFiles(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})
}

// DownloadFile downloads a file from remote to local
func (s *SFTPClient) DownloadFile(remotePath, localPath string) error {
	if s.sftpClient == nil {
		return fmt.Errorf("SFTP client not connected")
	}

	// Open remote file
	remoteFile, err := s.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy data
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// UploadFile uploads a file from local to remote
func (s *SFTPClient) UploadFile(localPath, remotePath string) error {
	if s.sftpClient == nil {
		return fmt.Errorf("SFTP client not connected")
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// Create remote file
	remoteFile, err := s.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Copy data
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// CreateFile creates a new empty file
func (s *SFTPClient) CreateFile(path string) error {
	if s.sftpClient == nil {
		return fmt.Errorf("SFTP client not connected")
	}

	file, err := s.sftpClient.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return nil
}

// RenameFile renames a file
func (s *SFTPClient) RenameFile(oldPath, newPath string) error {
	if s.sftpClient == nil {
		return fmt.Errorf("SFTP client not connected")
	}

	err := s.sftpClient.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// CreateLocalFile creates a new empty file locally
func CreateLocalFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()
	return nil
}

// RenameLocalFile renames a local file
func RenameLocalFile(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to rename local file: %w", err)
	}
	return nil
}

// CreateLocalDirAndFile creates directory structure and file locally
func CreateLocalDirAndFile(dir, filePath string) error {
	// Create directories if they don't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return nil
}

// CreateDirAndFile creates directory structure and file remotely
func (s *SFTPClient) CreateDirAndFile(dir, filePath string) error {
	if s.sftpClient == nil {
		return fmt.Errorf("SFTP client not connected")
	}

	// Create directories recursively
	if err := s.sftpClient.MkdirAll(dir); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create the file
	file, err := s.sftpClient.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return nil
}

// DeleteLocalFile deletes a local file or directory
func DeleteLocalFile(path string, isDir bool) error {
	if isDir {
		err := os.RemoveAll(path)
		if err != nil {
			return fmt.Errorf("failed to delete directory: %w", err)
		}
	} else {
		err := os.Remove(path)
		if err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}
	return nil
}

// DeleteFile deletes a remote file or directory
func (s *SFTPClient) DeleteFile(path string, isDir bool) error {
	if s.sftpClient == nil {
		return fmt.Errorf("SFTP client not connected")
	}

	if isDir {
		// Remove directory recursively
		err := s.removeDir(path)
		if err != nil {
			return fmt.Errorf("failed to delete directory: %w", err)
		}
	} else {
		err := s.sftpClient.Remove(path)
		if err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}
	return nil
}

// removeDir recursively removes a directory
func (s *SFTPClient) removeDir(path string) error {
	// List directory contents
	entries, err := s.sftpClient.ReadDir(path)
	if err != nil {
		return err
	}

	// Remove all contents first
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if err := s.removeDir(entryPath); err != nil {
				return err
			}
		} else {
			if err := s.sftpClient.Remove(entryPath); err != nil {
				return err
			}
		}
	}

	// Remove the directory itself
	return s.sftpClient.RemoveDirectory(path)
}
