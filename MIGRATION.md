# Migration Guide: JSON to SSH Config

## Overview

Starting from version 1.1.0, SSH-X-Term now stores connection information in the standard `~/.ssh/config` file instead of a custom JSON file. This makes your SSH connections compatible with standard SSH tools and provides better integration with your existing SSH workflow.

## Important Changes in v1.1.0

### Clean Initialization
When you first run v1.1.0, SSH-X-Term will:

1. **Create dated backups** of your `~/.ssh/config` file (e.g., `config.backup.20231124-143052`)
2. **Read all existing SSH entries** (even those without sxt metadata)
3. **Add sxt metadata** to all entries as comments
4. **Rewrite the config** with all entries properly tagged
5. **Attempt password recovery** from keyring using hostname as fallback

### No Duplicates
The new implementation ensures that:
- ✅ Existing SSH config entries are **never duplicated**
- ✅ Each entry gets a unique `#sxt:id` on first save
- ✅ All entries are properly managed after first run
- ✅ Dated backups are created before every save operation

## What Changed?

### Before (v1.0.x)
- Connections stored in `~/.config/ssh-x-term/ssh-x-term.json`
- Custom JSON format
- Passwords stored in system keyring

### After (v1.1.0+)
- Connections stored in `~/.ssh/config`
- Standard SSH config format with metadata comments
- Passwords still stored in system keyring
- Fully compatible with standard SSH command-line tools
- **Dated backups** created automatically (e.g., `config.backup.YYYYMMDD-HHMMSS`)

## Automatic Migration

When you first run the new version, SSH-X-Term will automatically:

1. Detect your old JSON configuration file (if exists)
2. Migrate all connections to `~/.ssh/config`
3. Preserve passwords in the system keyring
4. Create a backup of your old config at `~/.config/ssh-x-term/ssh-x-term.json.migrated`

**For existing SSH config entries:**
- Creates dated backup before any modifications
- Adds sxt metadata to all entries
- Attempts to recover passwords from keyring using:
  1. Existing connection ID
  2. Host pattern
  3. Hostname
  4. `username@hostname` combination

## Password Recovery

If you already had SSH connections before upgrading, SSH-X-Term will try to find passwords in your keyring:

1. **If found**: Password is automatically associated with the new connection ID
2. **If not found**: You'll be prompted to enter the password when connecting

The following keyring IDs are checked:
- Original connection ID (from JSON config)
- SSH Host pattern (e.g., "myserver")
- Hostname (e.g., "example.com")
- User@Host combination (e.g., "admin@example.com")

## Manual Migration

If automatic migration was skipped, you can manually migrate:

1. **Backup your current SSH config:**
   ```bash
   cp ~/.ssh/config ~/.ssh/config.backup
   ```

2. **View your old connections:**
   Your old connections are in `~/.config/ssh-x-term/ssh-x-term.json`

3. **Add connections through SSH-X-Term:**
   - Run `sxt`
   - Choose "Local Storage"
   - Add your connections manually through the TUI

## SSH Config Format

SSH-X-Term stores connections in standard SSH config format with special comments for metadata:

```ssh
#sxt:id=sxt-a1b2c3d4e5f6g7h8
#sxt:name=My Production Server
#sxt:notes=Production database server
#sxt:use_password=true
Host production-db
    HostName db.example.com
    Port 22
    User admin
```

### Metadata Comments

- `#sxt:id` - Unique identifier for the connection
- `#sxt:name` - Display name in SSH-X-Term
- `#sxt:notes` - Additional notes
- `#sxt:use_password` - Whether to use password or key authentication
- `#sxt:public_key` - Public key content (if applicable)
- `#sxt:organization_id` - Bitwarden organization ID (if using Bitwarden)

## Benefits of SSH Config Storage

1. **Standard Compatibility:** Your connections work with regular `ssh` command
2. **Better Integration:** Share connections across multiple tools
3. **Familiar Format:** Standard SSH config syntax
4. **Easy Editing:** Edit connections manually if needed
5. **Version Control:** Easily version control your SSH config (without passwords)

## Troubleshooting

### Migration Failed

If migration fails:
1. Check the logs at `~/.config/ssh-x-term/sxt.log`
2. Ensure `~/.ssh` directory exists and is writable
3. Manually add connections through the TUI

### Lost Connections

If connections are missing after migration:
1. Check your old config at `~/.config/ssh-x-term/ssh-x-term.json.migrated`
2. Check `~/.ssh/config` for entries
3. Re-add connections through SSH-X-Term TUI

### Password Issues

If passwords are not working:
- Passwords remain in the system keyring
- The keyring service name hasn't changed
- Try re-entering passwords if needed

## Reverting to Old Version

If you need to revert:

1. **Restore your old JSON config:**
   ```bash
   cp ~/.config/ssh-x-term/ssh-x-term.json.migrated ~/.config/ssh-x-term/ssh-x-term.json
   ```

2. **Install old version:**
   ```bash
   npm install -g ssh-x-term@1.0.23
   ```

3. **Remove SSH config entries** (optional):
   Edit `~/.ssh/config` and remove lines starting with `#sxt:`

## Questions?

If you encounter issues during migration:
- Check the [GitHub Issues](https://github.com/eugeniofciuvasile/ssh-x-term/issues)
- Open a new issue with details about your setup
- Include logs from `~/.config/ssh-x-term/sxt.log`
