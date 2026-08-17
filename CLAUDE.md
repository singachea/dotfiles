# CLAUDE.md - Dotfiles Setup Instructions

This file provides guidance to Claude Code when setting up dotfiles on a new machine.

## Purpose

This repository is the kit for AI coding-tool config. The `kit` CLI
(`cmd/kit`) reads live agent homes and explains drift. Do not invent a
second sync tool; extend `kit`.

When the user asks for status of plugins/skills across machines, run:

```bash
~/dotfiles/bin/kit status --root ~/dotfiles
# or kit board --no-open
```

## Repository Structure

```
~/dotfiles/
├── cmd/kit/                     # kit CLI
├── internal/kit/                # scan, findings, text, board
├── claude/                      # legacy Claude settings copies
├── CLAUDE.md
└── README.md
```

## What's Tracked vs Ignored

**✅ Tracked (safe to sync):**
- `claude/settings.json` - Global settings, enabled plugins
- `claude/settings.local.json` - Auto-allowed permissions (no sensitive data)

**❌ Ignored (sensitive/machine-specific):**
- `claude/anthropic_key.sh` - API keys (security risk)
- `claude/history.jsonl` - Conversation history (privacy)
- `claude/.claude.json` - Session data
- `claude/projects/` - Project-specific state
- `claude/cache/`, `claude/session-env/`, etc. - Temporary files

## Setup on New Machine

When the user asks you to "set up dotfiles" or "sync settings", follow these steps:

### Step 1: Clone the repository (if not already cloned)

```bash
git clone https://github.com/singachea/dotfiles.git ~/dotfiles
```

### Step 2: Backup existing settings (if they exist)

```bash
# Check if settings exist
if [ -f ~/.claude/settings.json ]; then
    cp ~/.claude/settings.json ~/.claude/settings.json.backup
    echo "✓ Backed up existing settings.json"
fi

if [ -f ~/.claude/settings.local.json ]; then
    cp ~/.claude/settings.local.json ~/.claude/settings.local.json.backup
    echo "✓ Backed up existing settings.local.json"
fi
```

### Step 3: Create symlinks

```bash
# Remove existing files (backups already created)
rm -f ~/.claude/settings.json ~/.claude/settings.local.json

# Create symlinks to dotfiles repo
ln -sf ~/dotfiles/claude/settings.json ~/.claude/settings.json
ln -sf ~/dotfiles/claude/settings.local.json ~/.claude/settings.local.json

echo "✓ Symlinks created"
```

### Step 4: Verify symlinks

```bash
ls -la ~/.claude/settings*.json
```

You should see output like:
```
lrwxr-xr-x ... /Users/singachea/.claude/settings.json -> /Users/singachea/dotfiles/claude/settings.json
lrwxr-xr-x ... /Users/singachea/.claude/settings.local.json -> /Users/singachea/dotfiles/claude/settings.local.json
```

### Step 5: Inform the user

Tell the user:
- ✓ Dotfiles setup complete
- Settings are now synced via Git
- They should restart Claude Code to apply changes
- Backups are saved at `~/.claude/settings*.backup` (if they existed)

## Updating Settings

When the user modifies settings and wants to sync across machines:

### On the machine where changes were made:

```bash
cd ~/dotfiles
git add claude/
git commit -m "Update Claude Code settings"
git push
```

### On other machines:

```bash
cd ~/dotfiles
git pull
```

Changes apply immediately via symlinks (no need to copy files).

## Adding New Config Files

If the user wants to add more dotfiles (e.g., `.zshrc`, `.gitconfig`):

1. **Copy the file to the repo:**
   ```bash
   cp ~/.zshrc ~/dotfiles/.zshrc
   ```

2. **Create a symlink:**
   ```bash
   rm ~/.zshrc
   ln -sf ~/dotfiles/.zshrc ~/.zshrc
   ```

3. **Commit and push:**
   ```bash
   cd ~/dotfiles
   git add .zshrc
   git commit -m "Add .zshrc"
   git push
   ```

4. **Update .gitignore if needed** - Add patterns for sensitive data

## Current Settings Summary

### settings.json
- API key helper: `~/.claude/anthropic_key.sh`
- Enabled plugins: `rust-analyzer-lsp@claude-plugins-official`

### settings.local.json
Pre-approved permissions (no prompts):
- `Bash` - All shell commands
- `Read` - Read files
- `Glob` - File pattern matching
- `Grep` - Content search
- `Edit` - Edit existing files
- `Write` - Create new files
- `Task` - Launch background agents
- `WebFetch` - Fetch web content
- `WebSearch` - Search the web

## Troubleshooting

**Symlinks not working?**
```bash
# Verify symlinks exist
ls -la ~/.claude/settings*.json

# Recreate if broken
rm ~/.claude/settings.json ~/.claude/settings.local.json
ln -sf ~/dotfiles/claude/settings.json ~/.claude/settings.json
ln -sf ~/dotfiles/claude/settings.local.json ~/.claude/settings.local.json
```

**Settings not syncing?**
```bash
# Pull latest changes
cd ~/dotfiles
git pull

# Verify file contents
cat ~/.claude/settings.json
```

**Want to stop syncing?**
```bash
# Remove symlinks
rm ~/.claude/settings.json ~/.claude/settings.local.json

# Restore from backup or create new files
cp ~/.claude/settings.json.backup ~/.claude/settings.json
```

## GitHub Repository

- **URL:** https://github.com/singachea/dotfiles
- **Owner:** singachea
- **Branch:** mainline

## User Preferences

- Username: singachea
- Dotfiles location: `~/dotfiles`
- Settings synced: Claude Code configuration
- Permissions: All read/write tools auto-approved
