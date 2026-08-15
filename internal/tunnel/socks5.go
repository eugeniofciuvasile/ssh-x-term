package tunnel

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

const (
	socks5Version = 0x05

	// Auth methods
	socks5AuthNone = 0x00

	// Commands
	socks5CmdConnect = 0x01

	// Address types
	socks5AtypIPv4   = 0x01
	socks5AtypDomain = 0x03
	socks5AtypIPv6   = 0x04

	// Reply codes
	socks5RepSuccess          = 0x00
	socks5RepGeneralFailure   = 0x01
	socks5RepCmdNotSupported  = 0x07
	socks5RepAtypNotSupported = 0x08
)

// HandleSOCKS5Conn handles a SOCKS5 client connection and forwards it through the SSH client
func HandleSOCKS5Conn(clientConn net.Conn, sshClient *ssh.Client) (net.Conn, error) {
	// 1. Negotiation phase
	// Read version and number of methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, header); err != nil {
		return nil, fmt.Errorf("failed to read SOCKS5 header: %w", err)
	}

	if header[0] != socks5Version {
		return nil, fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(clientConn, methods); err != nil {
		return nil, fmt.Errorf("failed to read SOCKS5 auth methods: %w", err)
	}

	// Reply with NO AUTHENTICATION REQUIRED
	if _, err := clientConn.Write([]byte{socks5Version, socks5AuthNone}); err != nil {
		return nil, fmt.Errorf("failed to write SOCKS5 auth reply: %w", err)
	}

	// 2. Request phase
	reqHeader := make([]byte, 4)
	if _, err := io.ReadFull(clientConn, reqHeader); err != nil {
		return nil, fmt.Errorf("failed to read SOCKS5 request: %w", err)
	}

	if reqHeader[0] != socks5Version {
		return nil, fmt.Errorf("invalid SOCKS version in request: %d", reqHeader[0])
	}

	if reqHeader[1] != socks5CmdConnect {
		// Send command not supported
		_, _ = clientConn.Write([]byte{socks5Version, socks5RepCmdNotSupported, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return nil, fmt.Errorf("unsupported SOCKS5 command: %d (only CONNECT is supported)", reqHeader[1])
	}

	var destAddr string
	switch reqHeader[3] {
	case socks5AtypIPv4:
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(clientConn, ipv4); err != nil {
			return nil, fmt.Errorf("failed to read IPv4 address: %w", err)
		}
		destAddr = net.IP(ipv4).String()

	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(clientConn, lenBuf); err != nil {
			return nil, fmt.Errorf("failed to read domain length: %w", err)
		}
		domainLen := int(lenBuf[0])
		domainBuf := make([]byte, domainLen)
		if _, err := io.ReadFull(clientConn, domainBuf); err != nil {
			return nil, fmt.Errorf("failed to read domain: %w", err)
		}
		destAddr = string(domainBuf)

	case socks5AtypIPv6:
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(clientConn, ipv6); err != nil {
			return nil, fmt.Errorf("failed to read IPv6 address: %w", err)
		}
		destAddr = net.IP(ipv6).String()

	default:
		_, _ = clientConn.Write([]byte{socks5Version, socks5RepAtypNotSupported, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return nil, errors.New("unsupported address type")
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, portBuf); err != nil {
		return nil, fmt.Errorf("failed to read destination port: %w", err)
	}
	destPort := binary.BigEndian.Uint16(portBuf)

	target := net.JoinHostPort(destAddr, strconv.Itoa(int(destPort)))

	// Dial target through SSH connection
	targetConn, err := sshClient.Dial("tcp", target)
	if err != nil {
		_, _ = clientConn.Write([]byte{socks5Version, socks5RepGeneralFailure, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
		return nil, fmt.Errorf("failed to dial target %s via SSH: %w", target, err)
	}

	// Reply SUCCESS
	_, err = clientConn.Write([]byte{socks5Version, socks5RepSuccess, 0x00, socks5AtypIPv4, 0, 0, 0, 0, 0, 0})
	if err != nil {
		_ = targetConn.Close()
		return nil, fmt.Errorf("failed to write SOCKS5 success reply: %w", err)
	}

	return targetConn, nil
}
