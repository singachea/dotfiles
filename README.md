# Dotfiles

Personal configuration files synced across machines.

## Claude Code Settings

### Setup on a new machine

```bash
# Clone this repo
git clone https://github.com/singachea/dotfiles.git ~/dotfiles

# Backup existing settings (optional)
cp ~/.claude/settings.json ~/.claude/settings.json.backup
cp ~/.claude/settings.local.json ~/.claude/settings.local.json.backup

# Create symlinks to sync settings
ln -sf ~/dotfiles/claude/settings.json ~/.claude/settings.json
ln -sf ~/dotfiles/claude/settings.local.json ~/.claude/settings.local.json
```

### Files included

- `claude/settings.json` - Global Claude Code settings (plugins, API helper)
- `claude/settings.local.json` - Permissions and local overrides

### What's NOT synced

For security and privacy, these files are NOT tracked:
- API keys (`anthropic_key.sh`)
- Conversation history (`history.jsonl`)
- Session data (`.claude.json`)
- Project-specific state (`projects/`)
- Cache and temporary files

### Updating settings

After making changes on any machine:

```bash
cd ~/dotfiles
git add claude/
git commit -m "Update Claude Code settings"
git push
```

On other machines:

```bash
cd ~/dotfiles
git pull
```

Changes will sync automatically via symlinks.
