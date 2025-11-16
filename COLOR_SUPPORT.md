# Terminal Color Support

## Overview

The SSH-X-Term terminal emulator now fully supports ANSI/VT100 color escape sequences, allowing colored output from SSH sessions to be displayed correctly within the Bubble Tea UI.

## Supported Features

### Color Modes

1. **8-Color Mode (Standard Colors)**
   - Foreground: ESC[30-37m (black, red, green, yellow, blue, magenta, cyan, white)
   - Background: ESC[40-47m

2. **16-Color Mode (Bright Colors)**
   - Bright Foreground: ESC[90-97m
   - Bright Background: ESC[100-107m

3. **256-Color Mode**
   - Foreground: ESC[38;5;Nm (where N is 0-255)
   - Background: ESC[48;5;Nm

### Text Attributes

- **Bold**: ESC[1m / ESC[22m (not bold)
- **Reverse Video**: ESC[7m / ESC[27m (not reverse)
- **Reset**: ESC[0m (resets all attributes)

## Implementation Details

### Data Structure

The terminal buffer stores each cell as a `cell` structure containing:
- `char`: The character (rune)
- `attrs`: Cell attributes (colors, bold, reverse)

```go
type cell struct {
    char  rune
    attrs cellAttrs
}

type cellAttrs struct {
    fgColor int  // -1 for default, 0-255 for colors
    bgColor int  // -1 for default, 0-255 for colors
    bold    bool
    reverse bool
}
```

### Rendering

The `Render()` function applies ANSI color codes based on stored attributes:
- Colors are applied only when attributes change (optimization)
- End-of-line resets attributes to default
- Cursor position uses reverse video

## Examples

### Shell Prompts
```bash
PS1='\[\e[32m\]\u\[\e[0m\]@\[\e[34m\]\h\[\e[0m\]:\[\e[36m\]\w\[\e[0m\]\$ '
```
This creates a colored prompt like: `user@host:~/path$` with green username, blue hostname, and cyan path.

### ls with Colors
When using `ls --color=auto`, directories appear in blue, executables in green, etc.

### Syntax Highlighting
Tools like `bat`, `diff --color`, `grep --color` will display colored output correctly.

### Build Output
Build systems and CI tools that use colors (like npm, cargo, go test -v) will display properly.

## Testing

Run the test suite to verify color support:
```bash
go test ./internal/ui/components -v -run TestVTerminalColor
```

## Technical Notes

- The SSH session requests a `xterm-256color` PTY, enabling full color support from the remote shell
- Color attributes are stored per-cell in the buffer, preserving them through scrollback
- The implementation optimizes by only emitting color codes when attributes change
- RGB color mode (ESC[38;2;R;G;Bm) is parsed but not fully implemented (simplified to 256-color)

## Compatibility

The color implementation is compatible with:
- Standard ANSI/VT100 terminals
- xterm-256color
- Modern terminal emulators (Terminal.app, iTerm2, GNOME Terminal, Windows Terminal, etc.)
