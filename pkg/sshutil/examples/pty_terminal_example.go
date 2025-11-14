// Package main demonstrates the usage of the new PTY terminal functionality
// in the pkg/sshutil package.
//
// This example shows how to integrate the PTY terminal with SSH sessions
// to provide a fully functional terminal shell with scrollback, mouse support,
// signal handling, and proper environment setup.
//
//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
)

// Example 1: Basic PTY Terminal Usage
// This example shows the minimal code required to create and start a PTY terminal.
func exampleBasicUsage() {
	fmt.Println("Example 1: Basic PTY Terminal Usage")
	fmt.Println("This example demonstrates creating a PTY terminal with default settings.")
	fmt.Println("Note: This example requires a TTY to run properly.")
	fmt.Println()

	// In a real application, these would be connected to an SSH session
	// For demonstration purposes, we're just showing the API
	
	// Example I/O setup (in practice, these come from ssh.Session)
	// stdin, stdout, stderr := getSSHSessionStreams()
	
	fmt.Println("Creating PTY terminal with default options...")
	
	// Create terminal with default options
	// terminal, err := sshutil.NewPTYTerminal(stdin, stdout, stderr, nil)
	// if err != nil {
	//     log.Fatalf("Failed to create PTY terminal: %v", err)
	// }
	// defer terminal.Close()
	
	// Start the terminal (blocks until session ends)
	// if err := terminal.Start(); err != nil {
	//     log.Printf("Terminal session error: %v", err)
	// }
	
	fmt.Println("PTY terminal would be running here...")
	fmt.Println()
}

// Example 2: Advanced PTY Terminal with Custom Options
// This example shows how to configure the PTY terminal with custom options.
func exampleAdvancedUsage() {
	fmt.Println("Example 2: Advanced PTY Terminal with Custom Options")
	fmt.Println("This example demonstrates creating a PTY terminal with custom configuration.")
	fmt.Println()

	// Configure custom options
	opts := &sshutil.PTYTerminalOptions{
		Shell: "/bin/zsh", // Use zsh instead of default shell
		Environment: map[string]string{
			"EDITOR":    "vim",
			"LANG":      "en_US.UTF-8",
			"LC_ALL":    "en_US.UTF-8",
			"MY_VAR":    "custom_value",
		},
		ScrollbackLines: 20000,  // Larger scrollback buffer
		EnableMouse:     true,   // Enable mouse support
		Debug:           true,   // Enable debug logging
	}

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Shell: %s\n", opts.Shell)
	fmt.Printf("  Scrollback Lines: %d\n", opts.ScrollbackLines)
	fmt.Printf("  Mouse Support: %v\n", opts.EnableMouse)
	fmt.Printf("  Debug Mode: %v\n", opts.Debug)
	fmt.Printf("  Environment Variables: %d custom vars\n", len(opts.Environment))
	fmt.Println()

	// In practice:
	// terminal, err := sshutil.NewPTYTerminal(stdin, stdout, stderr, opts)
	// if err != nil {
	//     log.Fatalf("Failed to create PTY terminal: %v", err)
	// }
	// defer terminal.Close()
	
	// Set additional environment variable after creation
	// terminal.SetEnvironment("ADDITIONAL_VAR", "value")
	
	// Start the terminal
	// if err := terminal.Start(); err != nil {
	//     log.Printf("Terminal session error: %v", err)
	// }
	
	fmt.Println("Advanced PTY terminal would be running here...")
	fmt.Println()
}

// Example 3: Scrollback Buffer Usage
// This example shows how to work with the scrollback buffer.
func exampleScrollbackUsage() {
	fmt.Println("Example 3: Scrollback Buffer Usage")
	fmt.Println("This example demonstrates accessing and using the scrollback buffer.")
	fmt.Println()

	// Create a scrollback buffer
	scrollback := sshutil.NewScrollbackBuffer(1000)

	// Add some sample lines
	scrollback.AddLine([]byte("Line 1: System initialized"))
	scrollback.AddLine([]byte("Line 2: User logged in"))
	scrollback.AddLine([]byte("Line 3: Running command..."))
	scrollback.AddLine([]byte("Line 4: Command completed"))

	// Get all lines from the buffer
	lines := scrollback.GetLines()
	fmt.Printf("Scrollback buffer contains %d lines:\n", len(lines))
	for i, line := range lines {
		fmt.Printf("  [%d] %s\n", i+1, string(line))
	}
	fmt.Println()

	// Clear the buffer
	scrollback.Clear()
	lines = scrollback.GetLines()
	fmt.Printf("After clearing, buffer contains %d lines\n", len(lines))
	fmt.Println()

	// In practice, with a PTY terminal:
	// terminal, _ := sshutil.NewPTYTerminal(stdin, stdout, stderr, nil)
	// scrollback := terminal.GetScrollback()
	// lines := scrollback.GetLines()
	// Process the lines...
}

// Example 4: Environment Variable Management
// This example shows how to manage environment variables.
func exampleEnvironmentManagement() {
	fmt.Println("Example 4: Environment Variable Management")
	fmt.Println("This example demonstrates managing environment variables for the terminal.")
	fmt.Println()

	// Initial environment
	initialEnv := map[string]string{
		"SHELL":  "/bin/bash",
		"EDITOR": "nano",
		"PAGER":  "less",
	}

	_ = &sshutil.PTYTerminalOptions{
		Environment: initialEnv,
	}

	fmt.Println("Initial environment variables:")
	for k, v := range initialEnv {
		fmt.Printf("  %s=%s\n", k, v)
	}
	fmt.Println()

	// In practice:
	// terminal, _ := sshutil.NewPTYTerminal(stdin, stdout, stderr, opts)
	
	// Add more environment variables
	// terminal.SetEnvironment("GIT_EDITOR", "vim")
	// terminal.SetEnvironment("TERM", "xterm-256color")
	
	// Get all environment variables
	// env := terminal.GetEnvironment()
	// for k, v := range env {
	//     fmt.Printf("%s=%s\n", k, v)
	// }

	fmt.Println("Additional variables would be set here...")
	fmt.Println()
}

// Example 5: Terminal Size Information
// This example shows how to get terminal size information.
func exampleTerminalSize() {
	fmt.Println("Example 5: Terminal Size Information")
	fmt.Println("This example demonstrates getting terminal size information.")
	fmt.Println()

	// Get current terminal size
	width, height, err := sshutil.GetTerminalSize()
	if err != nil {
		fmt.Printf("Unable to get terminal size: %v\n", err)
		fmt.Println("(This is expected if not running in a terminal)")
	} else {
		fmt.Printf("Current terminal size: %dx%d (width x height)\n", width, height)
	}
	fmt.Println()

	// In practice, with a PTY terminal:
	// terminal, _ := sshutil.NewPTYTerminal(stdin, stdout, stderr, nil)
	// width, height := terminal.GetSize()
	// fmt.Printf("Terminal size: %dx%d\n", width, height)
	
	// The terminal automatically handles SIGWINCH (resize) events
	// and updates its internal size accordingly
}

// Example 6: Integration with SSH Sessions (Conceptual)
// This example shows how the PTY terminal integrates with SSH sessions.
func exampleSSHIntegration() {
	fmt.Println("Example 6: Integration with SSH Sessions")
	fmt.Println("This example demonstrates integrating PTY terminal with SSH.")
	fmt.Println()

	fmt.Println("Conceptual code for SSH integration:")
	fmt.Println()
	fmt.Print("  // Create SSH session\n")
	fmt.Print("  sshSession, err := ssh.NewSession(connConfig)\n")
	fmt.Print("  if err != nil {\n")
	fmt.Print("      log.Fatalf(\"SSH connection failed: %v\", err)\n")
	fmt.Print("  }\n")
	fmt.Print("  defer sshSession.Close()\n")
	fmt.Println()
	fmt.Print("  // Create PTY terminal using SSH I/O streams\n")
	fmt.Print("  terminal, err := sshutil.NewPTYTerminal(\n")
	fmt.Print("      sshSession.Stdin(),\n")
	fmt.Print("      sshSession.Stdout(),\n")
	fmt.Print("      sshSession.Stderr(),\n")
	fmt.Print("      &sshutil.PTYTerminalOptions{\n")
	fmt.Print("          EnableMouse:     true,\n")
	fmt.Print("          ScrollbackLines: 10000,\n")
	fmt.Print("          Debug:           false,\n")
	fmt.Print("      },\n")
	fmt.Print("  )\n")
	fmt.Print("  if err != nil {\n")
	fmt.Print("      log.Fatalf(\"Terminal creation failed: %v\", err)\n")
	fmt.Print("  }\n")
	fmt.Print("  defer terminal.Close()\n")
	fmt.Println()
	fmt.Print("  // Start the terminal (blocks until session ends)\n")
	fmt.Print("  if err := terminal.Start(); err != nil {\n")
	fmt.Print("      log.Printf(\"Terminal error: %v\", err)\n")
	fmt.Print("  }\n")
	fmt.Println()
}

// Example 7: Features Overview
// This example provides an overview of all PTY terminal features.
func exampleFeaturesOverview() {
	fmt.Println("Example 7: PTY Terminal Features Overview")
	fmt.Println("==========================================")
	fmt.Println()
	
	fmt.Println("✓ Core Terminal Shell Functionality:")
	fmt.Println("  - PTY management for interactive shell sessions")
	fmt.Println("  - Raw mode terminal support")
	fmt.Println("  - I/O streaming with proper buffering")
	fmt.Println()
	
	fmt.Println("✓ Exit Functionality:")
	fmt.Println("  - EOF detection (Ctrl+D on Unix, Ctrl+D/Ctrl+Z on Windows)")
	fmt.Println("  - Graceful session termination")
	fmt.Println("  - Proper cleanup on exit")
	fmt.Println()
	
	fmt.Println("✓ Scrolling Support:")
	fmt.Println("  - Configurable scrollback buffer (default 10,000 lines)")
	fmt.Println("  - Line-by-line output capture")
	fmt.Println("  - Thread-safe buffer access")
	fmt.Println()
	
	fmt.Println("✓ Mouse Integration (Unix/Linux/macOS):")
	fmt.Println("  - X10 compatibility mode")
	fmt.Println("  - Mouse button event tracking")
	fmt.Println("  - SGR extended mouse mode")
	fmt.Println("  - Text selection and copy-paste support")
	fmt.Println()
	
	fmt.Println("✓ Signal Handling:")
	fmt.Println("  - SIGWINCH: Automatic window resize detection")
	fmt.Println("  - SIGINT (Ctrl+C): Forwarded to remote session")
	fmt.Println("  - SIGTERM: Clean shutdown")
	fmt.Println()
	
	fmt.Println("✓ Environment Setup:")
	fmt.Println("  - Default environment (TERM, PATH, HOME)")
	fmt.Println("  - Custom environment variables")
	fmt.Println("  - Dynamic variable updates")
	fmt.Println()
	
	fmt.Println("✓ Logging and Debugging:")
	fmt.Println("  - Optional debug mode")
	fmt.Println("  - Detailed operation logging")
	fmt.Println("  - Error tracking and reporting")
	fmt.Println()
	
	fmt.Println("✓ Cross-Platform Support:")
	fmt.Println("  - Unix/Linux/macOS: Full feature set")
	fmt.Println("  - Windows: Core features with platform adaptations")
	fmt.Println()
}

func main() {
	if len(os.Args) > 1 {
		// Run specific example if argument provided
		example := os.Args[1]
		switch example {
		case "1":
			exampleBasicUsage()
		case "2":
			exampleAdvancedUsage()
		case "3":
			exampleScrollbackUsage()
		case "4":
			exampleEnvironmentManagement()
		case "5":
			exampleTerminalSize()
		case "6":
			exampleSSHIntegration()
		case "7":
			exampleFeaturesOverview()
		default:
			fmt.Printf("Unknown example: %s\n", example)
			fmt.Println("Available examples: 1-7")
		}
		return
	}

	// Run all examples
	log.SetFlags(0) // Disable timestamp in logs for cleaner output

	fmt.Println("PTY Terminal Examples")
	fmt.Println("=====================")
	fmt.Println()

	exampleBasicUsage()
	exampleAdvancedUsage()
	exampleScrollbackUsage()
	exampleEnvironmentManagement()
	exampleTerminalSize()
	exampleSSHIntegration()
	exampleFeaturesOverview()

	fmt.Println("All examples completed!")
	fmt.Println()
	fmt.Println("Usage: go run examples/pty_terminal_example.go [1-7]")
	fmt.Println("  Run a specific example by number, or all examples if no argument provided.")
}
