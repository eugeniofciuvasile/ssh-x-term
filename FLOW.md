# SSH-X-Term Bubble Tea Architecture Schema

## Core Bubble Tea Loop

SSH-X-Term is structured around Bubble Tea’s Model-Update-View (MUV) architecture, with distinct application states and modular UI components.

### 1. Model

- **File:** `internal/ui/model.go`
- **Definition:** The `Model` struct holds all UI state, including:
  - `state`: Current high-level app state (`AppState` enum, e.g., `StateConnectionList`, `StateSSHTerminal`)
  - Pointers to all UI components (e.g., `connectionList`, `terminal`, `bitwardenLoginForm`, etc.)
  - Dimensions (`width`, `height`)
  - Configuration and credential managers

- **App States (`AppState` enum):**
  - `StateSelectStorage`
  - `StateBitwardenConfig`
  - `StateConnectionList`
  - `StateAddConnection`
  - `StateEditConnection`
  - `StateSSHTerminal`
  - `StateBitwardenLogin`
  - `StateBitwardenUnlock`
  - `StateOrganizationSelect`
  - `StateCollectionSelect`

- **Active Component:**  
  The method `getActiveComponent()` returns the UI component corresponding to the current state (e.g., `connectionList` for `StateConnectionList`).

---

### 2. Update

- **File:** `internal/ui/update.go`
- **Definition:**  
  The `Update(msg tea.Msg) (tea.Model, tea.Cmd)` method is the heart of the event loop.
  - Receives messages (keyboard, window size, etc.).
  - Switches on message type and current state.
  - Delegates to the active component’s `Update` method.
  - Handles global logic (e.g., state transitions, error handling, component resets).

- **Stateful Logic Examples:**
  - In `StateConnectionList`, handles keybindings:  
    - 'a': Add connection (`StateAddConnection`)
    - 'e': Edit highlighted connection (`StateEditConnection`)
    - 'd': Delete highlighted connection
    - 'o': Toggle open in new terminal
    - 'esc': Back to previous state
    - 'ctrl+c': Quit
  - For window resizing, updates model dimensions and resizes the active component.

- **Component Results:**  
  After delegating to a component’s `Update`, the main model processes any results (e.g., a submitted form advances to the next state).

---

### 3. View

- **File:** `internal/ui/view.go`
- **Definition:**  
  The `View() string` method renders the UI:
  - Prepends the app title.
  - Renders the active component’s view (from `getActiveComponent().View()`).
  - Appends contextual instructions depending on the state.
  - Displays error messages if present.

---

### 4. UI Components

Each major function of the app is encapsulated in a component (e.g., `ConnectionList`, `ConnectionForm`, `TerminalComponent`, `BitwardenLoginForm`, etc.), each implementing Bubble Tea’s `Model`, `Update`, and `View` methods.

- **Location:** `internal/ui/components/`
- **Component Examples:**
  - `ConnectionList`
  - `ConnectionForm`
  - `TerminalComponent`
  - `BitwardenLoginForm`
  - `BitwardenUnlockForm`
  - `BitwardenOrganizationList`
  - `BitwardenCollectionList`
  - `StorageSelect`

---

### 5. State Transitions

- Transitions are managed via the main model’s `state` field.
- Example flow:
  1. **Startup**:  
     - State: `StateSelectStorage`
     - User chooses storage backend (local/Bitwarden).
  2. **After Storage Selection**:  
     - State: `StateConnectionList` (local) or `StateBitwardenConfig/StateBitwardenLogin/StateBitwardenUnlock` (Bitwarden).
  3. **Connection Actions**:  
     - Add/Edit/Delete/Connect transitions handled via forms and connection list.
  4. **SSH Session**:  
     - State: `StateSSHTerminal`, which manages the live SSH session UI.

---

## Bubble Tea Loop Schema Diagram

```mermaid
flowchart TD
    Start["Start: runApp()"]
    Start --> Program["tea.NewProgram (Model, Options)"]
    Program --> Loop["Main Bubble Tea Loop"]
    Loop -->|msg: User/system events| Update["Update(msg)"]
    Update --> ActiveComponent["Active Component Update"]
    ActiveComponent --> HandleResult["Main Model handles result"]
    HandleResult --> View["View()"]
    View --> Render["Terminal UI Output"]
    Render --> Loop
```

---

## Summary Table: State → Component Mapping

| AppState                  | Active Component              | Description / UI |
|---------------------------|-------------------------------|------------------|
| StateSelectStorage        | StorageSelect                 | Storage backend selector |
| StateBitwardenConfig      | BitwardenConfigForm           | Bitwarden server/email config |
| StateBitwardenLogin       | BitwardenLoginForm            | Bitwarden login prompt |
| StateBitwardenUnlock      | BitwardenUnlockForm           | Bitwarden unlock prompt |
| StateOrganizationSelect   | BitwardenOrganizationList     | Org selector (Bitwarden) |
| StateCollectionSelect     | BitwardenCollectionList       | Collection selector (Bitwarden) |
| StateConnectionList       | ConnectionList                | SSH connections list |
| StateAddConnection        | ConnectionForm                | Add new SSH connection |
| StateEditConnection       | ConnectionForm                | Edit existing SSH connection |
| StateSSHTerminal          | TerminalComponent             | Interactive SSH terminal with VT100 emulation |

---

## Terminal Emulation Architecture

The SSH terminal is fully integrated within Bubble Tea using a custom virtual terminal emulator:

### Components

1. **VTerminal** (`internal/ui/components/vterm.go`)
   - Virtual terminal emulator with VT100/ANSI escape sequence parsing
   - Maintains display buffer and scrollback buffer (10,000 lines)
   - Handles cursor positioning, colors, and terminal control sequences
   - Supports text selection and clipboard operations

2. **BubbleTeaSession** (`internal/ssh/session_bubbletea_unix.go`, `session_bubbletea_windows.go`)
   - SSH session wrapper that works with Bubble Tea
   - Provides Read/Write interfaces for bidirectional communication
   - Handles window resize events
   - Platform-specific implementations for Unix and Windows

3. **TerminalComponent** (`internal/ui/components/terminal.go`)
   - Bubble Tea component that integrates VTerminal and BubbleTeaSession
   - Handles user input (keyboard and mouse)
   - Forwards keystrokes to SSH session
   - Renders terminal output within Bubble Tea UI
   - Manages scrolling and text selection

### Data Flow

```
User Input (Keyboard/Mouse)
    ↓
TerminalComponent.Update()
    ↓
BubbleTeaSession.Write() ──→ SSH Server
    ↓
SSH Server Output
    ↓
SSHOutputMsg
    ↓
VTerminal.Write() (ANSI parsing)
    ↓
VTerminal.Render()
    ↓
Display in Bubble Tea View
```

### Key Features

- **No Terminal Takeover**: Works entirely within Bubble Tea (no raw mode on host terminal)
- **Full Terminal Emulation**: Supports VT100/ANSI escape sequences
- **Scrollback**: 10,000 line buffer with keyboard and mouse scrolling
- **Text Selection**: Click and drag to select, automatic clipboard copy
- **Resize Support**: Handles terminal resize events seamlessly
- **Keyboard Support**: Full support for special keys (arrows, home, end, function keys, etc.)

---

---

## Application Startup and Configuration Flow

### Initial Launch Sequence

```mermaid
flowchart TD
    Start["main() Entry"] --> CheckTmux{"Running in tmux?"}
    CheckTmux -->|No| LaunchTmux["Launch new tmux session"]
    CheckTmux -->|Yes| InitConfig["Initialize ConfigManager"]
    LaunchTmux --> InitConfig
    InitConfig --> LoadConfig["Load configuration file"]
    LoadConfig --> CheckStorage{"Storage backend<br/>configured?"}
    CheckStorage -->|No| StateSelectStorage
    CheckStorage -->|Yes, Local| StateConnectionList
    CheckStorage -->|Yes, Bitwarden| CheckBWAuth{"Bitwarden<br/>authenticated?"}
    CheckBWAuth -->|Yes| StateConnectionList
    CheckBWAuth -->|No| StateBitwardenLogin
```

### Storage Selection Flow

When first launched or storage not configured:

1. **StateSelectStorage**: User chooses between:
   - **Local Storage**: Uses system keyring (go-keyring)
   - **Bitwarden**: Uses Bitwarden vault via CLI

2. **If Local Storage selected**:
   - Transitions directly to `StateConnectionList`
   - Credentials stored via go-keyring (Keychain/Secret Service/Credential Manager)

3. **If Bitwarden selected**:
   - Transitions to `StateBitwardenConfig` (if not configured)
   - Then `StateBitwardenLogin` (if not logged in)
   - Then `StateBitwardenUnlock` (if vault locked)
   - Finally to `StateConnectionList` or org/collection selection

---

## Detailed State Transition Flows

### Connection Management Flow

```mermaid
stateDiagram-v2
    [*] --> ConnectionList
    ConnectionList --> AddConnection: Press 'a'
    ConnectionList --> EditConnection: Press 'e'
    ConnectionList --> DeleteConnection: Press 'd'
    ConnectionList --> SSHTerminal: Press Enter
    ConnectionList --> SelectStorage: Press 's'
    
    AddConnection --> ConnectionList: Submit/Cancel
    EditConnection --> ConnectionList: Submit/Cancel
    DeleteConnection --> ConnectionList: Confirm
    
    SSHTerminal --> ConnectionList: Press Esc
    SelectStorage --> ConnectionList: Storage selected
```

### Bitwarden Authentication Flow

```mermaid
flowchart TD
    Start["Bitwarden Selected"] --> Config{"Config exists?"}
    Config -->|No| BWConfig["StateBitwardenConfig:<br/>Enter server URL & email"]
    Config -->|Yes| CheckStatus["Check Bitwarden status"]
    BWConfig --> CheckStatus
    
    CheckStatus --> CheckLogin{"Logged in?"}
    CheckLogin -->|No| BWLogin["StateBitwardenLogin:<br/>Enter master password"]
    CheckLogin -->|Yes| CheckUnlock{"Vault unlocked?"}
    
    BWLogin --> CheckUnlock
    CheckUnlock -->|No| BWUnlock["StateBitwardenUnlock:<br/>Enter master password"]
    CheckUnlock -->|Yes| CheckOrg{"Using organization?"}
    
    BWUnlock --> CheckOrg
    CheckOrg -->|Yes| SelectOrg["StateOrganizationSelect:<br/>Choose organization"]
    CheckOrg -->|No| ConnectionList["StateConnectionList"]
    
    SelectOrg --> SelectColl["StateCollectionSelect:<br/>Choose collection"]
    SelectColl --> ConnectionList
```

### SSH Connection Flow

```mermaid
sequenceDiagram
    participant User
    participant UI as TerminalComponent
    participant SSH as BubbleTeaSession
    participant Server as Remote SSH Server
    
    User->>UI: Press Enter on connection
    UI->>SSH: Initialize connection
    SSH->>Server: Establish SSH connection
    Server-->>SSH: Connection established
    SSH-->>UI: Session ready
    UI-->>User: Display terminal
    
    loop Interactive Session
        User->>UI: Type command
        UI->>SSH: Forward input
        SSH->>Server: Send data
        Server-->>SSH: Return output
        SSH-->>UI: SSHOutputMsg
        UI->>UI: VTerminal.Write (parse ANSI)
        UI-->>User: Render output
    end
    
    User->>UI: Press Esc
    UI->>SSH: Close session
    SSH->>Server: Disconnect
    UI-->>User: Return to connection list
```

---

## Component Interaction Patterns

### Model-Update-View Cycle with Components

```mermaid
flowchart LR
    User["User Input"] --> Msg["tea.Msg"]
    Msg --> Update["Model.Update()"]
    Update --> GetActive["getActiveComponent()"]
    GetActive --> CompUpdate["Component.Update()"]
    CompUpdate --> HandleResult["Handle component result"]
    HandleResult --> StateChange{"State change?"}
    StateChange -->|Yes| NewActive["Switch active component"]
    StateChange -->|No| UpdateModel["Update model state"]
    NewActive --> View["Model.View()"]
    UpdateModel --> View
    View --> CompView["Component.View()"]
    CompView --> Render["Rendered UI"]
    Render --> Display["Terminal Display"]
    Display --> User
```

### Credential Storage Architecture

```mermaid
flowchart TB
    subgraph "Storage Interface"
        Storage["Storage interface<br/>(Add, Delete, Get, List, Edit)"]
    end
    
    subgraph "Local Storage Backend"
        ConfigMgr["ConfigManager"] --> JSONFile["JSON Config File<br/>(metadata only)"]
        ConfigMgr --> Keyring["go-keyring<br/>(passwords/keys)"]
        Keyring --> OSKeyring["OS Keyring<br/>(Keychain/Secret Service/Credential Manager)"]
    end
    
    subgraph "Bitwarden Storage Backend"
        BWMgr["BitwardenManager"] --> BWCLI["Bitwarden CLI (bw)"]
        BWCLI --> BWVault["Bitwarden Vault<br/>(encrypted)"]
        BWCLI --> BWOrg["Organization/Collection"]
    end
    
    Storage --> ConfigMgr
    Storage --> BWMgr
    
    style Storage fill:#e1f5ff
    style ConfigMgr fill:#fff3e0
    style BWMgr fill:#fff3e0
```

---

## Message Flow and Async Operations

SSH-X-Term uses Bubble Tea's message passing for async operations:

### Async Message Types

1. **Connection Loading**:
   - `LoadConnectionsFinishedMsg`: Fired when connections loaded from storage
   
2. **Bitwarden Operations**:
   - `BitwardenStatusMsg`: Check if logged in/unlocked
   - `BitwardenLoadOrganizationsMsg`: Load available organizations
   - `BitwardenLoadCollectionsMsg`: Load collections for org
   - `BitwardenLoadConnectionsByCollectionMsg`: Load connections from collection
   - `BitwardenLoginResultMsg`: Login attempt result
   - `BitwardenUnlockResultMsg`: Unlock attempt result

3. **Connection Operations**:
   - `SaveConnectionResultMsg`: Save/update connection result
   - `DeleteConnectionResultMsg`: Delete connection result

4. **Terminal Operations**:
   - `SSHOutputMsg`: Output from SSH session (handled by TerminalComponent)
   - `tea.WindowSizeMsg`: Terminal resize event

### Error Handling Pattern

All async messages include an `Err` field. The main Update method checks for errors and:
1. Sets `model.errorMessage` if error occurred
2. Displays error in the View
3. Allows user to retry or cancel operation

---

## References

- [Bubble Tea Architecture](https://github.com/charmbracelet/bubbletea)
- [README](https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/README.md)
- [IMPLEMENTATION](https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/IMPLEMENTATION.md)
- Key Files:  
  - `internal/ui/model.go` - Main model with state machine
  - `internal/ui/update.go` - Event handling and state transitions
  - `internal/ui/view.go` - Rendering logic
  - `internal/ui/components/` - Reusable UI components
  - `internal/ui/components/vterm.go` - Virtual terminal emulator
  - `internal/ui/components/terminal.go` - Terminal component
  - `internal/ssh/session_bubbletea_unix.go` - Unix SSH session
  - `internal/ssh/session_bubbletea_windows.go` - Windows SSH session
  - `internal/config/storage.go` - Storage interface
  - `internal/config/config.go` - Local storage implementation
  - `internal/config/bitwarden.go` - Bitwarden storage implementation
