# Dotfiles / kit

Personal configuration for AI coding tools, plus `kit`, a Go CLI that
explains what is actually installed on this machine versus what the repo
knows about.

## Install kit

```bash
git clone git@github.com:singachea/dotfiles.git ~/dotfiles
cd ~/dotfiles && go install ./cmd/kit
```

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

Do not commit API keys, session history, plugin caches, or `auth.json`.
