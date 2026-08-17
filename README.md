# Dotfiles / kit

Personal configuration for AI coding tools, plus `kit`, a Go CLI that
explains what is actually installed on this machine versus what the repo
knows about.

## Install kit

Needs Go (`go version`). Do **not** use `go get` — that adds a library
dependency. For a CLI:

```bash
go install github.com/singachea/dotfiles/cmd/kit@mainline
```

That puts `kit` in `$(go env GOPATH)/bin` (usually `~/go/bin`). Put that
on your PATH if needed:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

`go install` only installs the binary. Skills, plugins, and settings live
in this repo, so a new machine still needs a checkout kit can find
(default `~/dotfiles`):

```bash
git clone git@github.com:singachea/dotfiles.git ~/dotfiles
```

If the repo is already at `~/dotfiles`, skip the clone. Then `kit` and
`kit doctor`.

`make install` still works if you prefer a local build into `~/bin/kit`.

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
