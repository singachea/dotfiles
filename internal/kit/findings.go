package kit

import (
	"fmt"
	"os"
	"strings"
)

func buildFindings(rep Report) []Finding {
	var f []Finding

	if rep.Profile == "" {
		f = append(f, Finding{
			Severity: SevInfo,
			Code:     "profile-unset",
			Title:    "This machine has no kit profile",
			Detail:   "New harvests will not know whether they belong in personal or work.",
			Why:      "kit profile personal|work writes ~/.config/kit/profile. Capture uses that as the default bucket.",
			Fix:      "kit profile personal   # or work",
		})
	}

	if rep.KitRoot == "" {
		f = append(f, Finding{
			Severity: SevError,
			Code:     "kit-root-missing",
			Title:    "Cannot find the kit repo",
			Detail:   "Looked at $KIT_ROOT, the current directory, and ~/dotfiles.",
			Why:      "Without a repo there is nothing to compare live agent homes against.",
			Fix:      "export KIT_ROOT=~/dotfiles   or run kit from the cloned repo",
		})
	}

	if rep.Git.Dirty {
		f = append(f, Finding{
			Severity: SevInfo,
			Code:     "git-dirty",
			Title:    "Kit repo has uncommitted changes",
			Detail:   fmt.Sprintf("%s (%s) is dirty", rep.Git.Branch, rep.Git.Head),
			Why:      "Status is comparing live homes to a working tree that is not what origin has.",
			Fix:      "cd " + rep.KitRoot + " && git status",
			Item:     rep.Git.Branch,
		})
	}

	for _, s := range rep.Settings {
		if s.Agent == "claude" && fileExists(s.Live) && !s.Linked {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "settings-not-linked",
				Title:    "Claude settings.json is a real file, not a kit symlink",
				Detail:   s.Live,
				Why:      "Claude writes plugins and hooks into this file. The copy in the repo is already stale on this machine.",
				Fix:      "kit will harvest this on capture; do not copy it by hand",
				Agent:    "claude",
				Item:     "settings.json",
			})
		}
		if s.Drift != "" {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "settings-drift",
				Title:    s.Agent + " " + s.File + " differs from the kit copy",
				Detail:   s.Drift,
				Why:      "The live file and the repo file disagree. The UI is the live source; the repo will stay wrong until capture.",
				Fix:      "kit capture   # after it is implemented, or inspect both files",
				Agent:    s.Agent,
				Item:     s.File,
			})
		}
	}

	// absolute paths in claude settings
	if live := settingLive(rep, "claude"); live != "" {
		for _, p := range findAbsPathsInFile(live) {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "absolute-path",
				Title:    "Claude settings contain a machine path",
				Detail:   p,
				Why:      "Hooks and status-line commands with /Users/... will fail on another computer.",
				Fix:      "Point the command at ~/... or a script inside the kit",
				Agent:    "claude",
				Item:     p,
			})
		}
	}

	for _, s := range rep.Skills {
		if s.Ignored {
			continue
		}
		switch s.Presence {
		case PresenceLiveOnly:
			agents := liveAgents(s)
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "skill-live-only",
				Title:    "Skill " + s.Name + " is not in the kit",
				Detail:   "Present on: " + strings.Join(agents, ", "),
				Why:      s.Why,
				Fix:      "kit capture will move this into skills/" + s.Name,
				Item:     s.Name,
			})
		case PresenceRepoOnly:
			f = append(f, Finding{
				Severity: SevInfo,
				Code:     "skill-repo-only",
				Title:    "Skill " + s.Name + " is only in the kit",
				Detail:   "No agent home has this skill yet.",
				Why:      s.Why,
				Fix:      "kit apply will symlink it into each agent",
				Item:     s.Name,
			})
		}
		if s.Diverges {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "skill-diverges",
				Title:    "Skill " + s.Name + " has different SKILL.md copies",
				Detail:   divergeDetail(s),
				Why:      "Agents have independent copies. One of them was edited and the others were not.",
				Fix:      "Decide which copy is canonical, then capture it",
				Item:     s.Name,
			})
		}
		for _, c := range s.Copies {
			if c.Kind == CopyBrokenLink {
				f = append(f, Finding{
					Severity: SevError,
					Code:     "skill-broken-link",
					Title:    "Broken skill symlink: " + s.Name + " on " + c.Agent,
					Detail:   c.Path + " → " + c.Target,
					Why:      "The agent will skip this skill.",
					Fix:      "kit apply",
					Agent:    c.Agent,
					Item:     s.Name,
				})
			}
		}
	}

	for _, p := range rep.Plugins {
		if !p.InManifest && (p.Enabled || p.Installed) {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "plugin-live-only",
				Title:    p.Agent + " plugin " + p.ID + " is not in the kit manifest",
				Detail:   p.Why,
				Why:      "Installing a plugin in the UI only updates that agent home. Other machines never see it.",
				Fix:      "kit capture will add it to manifest.yaml / profiles/" + orDefault(rep.Profile, "personal") + ".yaml",
				Agent:    p.Agent,
				Item:     p.ID,
			})
		}
		if p.Enabled && !p.Installed {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "plugin-missing-install",
				Title:    p.Agent + " plugin " + p.ID + " is enabled but not installed",
				Detail:   p.Why,
				Why:      "Settings ask for it; the install cache does not have it.",
				Fix:      "Reinstall with the agent's plugin command, or kit apply later",
				Agent:    p.Agent,
				Item:     p.ID,
			})
		}
	}

	for _, m := range rep.Marketplaces {
		if !m.Portable {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "marketplace-local-path",
				Title:    "Marketplace " + m.Name + " is a local directory",
				Detail:   m.Source,
				Why:      m.Why,
				Fix:      "Publish it as a git URL, or vendor it inside the kit as plugins/" + m.Name,
				Agent:    m.Agent,
				Item:     m.Name,
			})
		}
	}

	for _, m := range rep.MCP {
		if !m.Portable {
			f = append(f, Finding{
				Severity: SevWarn,
				Code:     "mcp-local-path",
				Title:    m.Agent + " MCP " + m.Name + " uses a machine path",
				Detail:   first(m.Command, m.URL),
				Why:      m.Why,
				Fix:      "Use a command on PATH or ${VAR} for secrets/paths",
				Agent:    m.Agent,
				Item:     m.Name,
			})
		}
	}

	for _, a := range rep.Agents {
		if a.ID == "grok" && a.Present {
			empty := true
			for _, s := range rep.Skills {
				for _, c := range s.Copies {
					if c.Agent == "grok" && c.Kind != CopyMissing {
						empty = false
					}
				}
			}
			if empty {
				f = append(f, Finding{
					Severity: SevInfo,
					Code:     "grok-empty-skills",
					Title:    "Grok has no skills in ~/.grok/skills",
					Detail:   "Grok still reads Claude and Cursor skill dirs when compat is on (the default).",
					Why:      "An empty Grok skills dir is not a bug. Status still lists those inherited skills under claude/cursor so you can see the real copies.",
					Fix:      "Optional: kit apply can also link into ~/.grok/skills so Grok does not depend on compat",
					Agent:    "grok",
				})
			}
		}
	}

	sortFindings(f)
	return f
}

func sortFindings(f []Finding) {
	rank := map[Severity]int{SevError: 0, SevWarn: 1, SevInfo: 2}
	for i := 0; i < len(f); i++ {
		for j := i + 1; j < len(f); j++ {
			if rank[f[j].Severity] < rank[f[i].Severity] ||
				(rank[f[j].Severity] == rank[f[i].Severity] && f[j].Code < f[i].Code) {
				f[i], f[j] = f[j], f[i]
			}
		}
	}
}

func liveAgents(s Skill) []string {
	var a []string
	for _, c := range s.Copies {
		if c.Kind != CopyMissing {
			a = append(a, c.Agent)
		}
	}
	return a
}

func divergeDetail(s Skill) string {
	var parts []string
	if s.RepoHash != "" {
		parts = append(parts, "kit="+s.RepoHash)
	}
	for _, c := range s.Copies {
		if c.Kind != CopyMissing && c.Hash != "" {
			parts = append(parts, c.Agent+"="+c.Hash)
		}
	}
	return strings.Join(parts, " ")
}

func settingLive(rep Report, agent string) string {
	for _, s := range rep.Settings {
		if s.Agent == agent {
			return s.Live
		}
	}
	return ""
}

func findAbsPathsInFile(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var found []string
	seen := map[string]bool{}
	for _, field := range strings.FieldsFunc(string(b), func(r rune) bool {
		return r == '"' || r == '\'' || r == ' ' || r == '\n' || r == '\t' || r == ','
	}) {
		if isAbsMachinePath(field) && !seen[field] {
			seen[field] = true
			found = append(found, field)
		}
	}
	return found
}

func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func summarize(rep Report) Summary {
	s := Summary{
		Skills:   len(rep.Skills),
		Plugins:  len(rep.Plugins),
		Findings: len(rep.Findings),
	}
	for _, a := range rep.Agents {
		if a.Present {
			s.AgentsPresent++
		}
	}
	for _, f := range rep.Findings {
		switch f.Severity {
		case SevError:
			s.Errors++
		case SevWarn:
			s.Warnings++
		default:
			s.Infos++
		}
	}
	return s
}
