# Dotfiles / kit

Personal configuration for AI coding tools, plus `kit`, a Go CLI that
explains what is actually installed on this machine versus what the repo
knows about.

## Install kit

Needs Go (`go version`). The binary lands at `~/bin/kit`.

**This machine already has the repo:**

```bash
cd ~/dotfiles && make install
```

**New machine:**

```bash
git clone git@github.com:singachea/dotfiles.git ~/dotfiles
cd ~/dotfiles && make install
```

If `kit` is not found, put `~/bin` on your PATH (zsh):

```bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Check: `kit` should print help. Then `kit profile personal` (or `work`) and `kit doctor`.

## Daily commands

```bash
kit doctor            # start here — pick a number to run that step
kit recommend         # same list, with the item names
kit fix               # copy-paste commands, in order
kit status            # full inventory
kit board             # tabbed HTML (Do next / Skills / Plugins / Settings)
kit remove casino-gaming-ui   # drop a skill from the kit (won't recapture)
kit capture --skills  # copy live skills into the kit repo
kit capture --plugins
kit capture --settings
kit profile personal  # or work
```

`apply` / `pull` / `push` / `sync` are not implemented yet.

`kit` never writes secrets. It reads live homes (`~/.claude`, `~/.grok`,
`~/.codex`, `~/.cursor`, `~/.config/opencode`, `~/.agents`, `~/.copilot`)
and compares them to this repo.

## Claude settings (legacy)

`claude/settings.json` is still in the repo. Live `~/.claude/settings.json`
is **not** a symlink and has already drifted — `kit status` reports that.

```bash
# optional, after you trust the repo copy
ln -sf ~/dotfiles/claude/settings.json ~/.claude/settings.json
ln -sf ~/dotfiles/claude/settings.local.json ~/.claude/settings.local.json
```

Do not commit API keys, session history, plugin caches, or `auth.json`.
