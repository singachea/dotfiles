package kit

import (
	"fmt"
	"strings"
)

func Recommend(rep Report) []Action {
	var next, later []Action

	add := func(a Action) {
		if a.Count == 0 {
			a.Count = len(a.Items)
		}
		if a.Later {
			later = append(later, a)
		} else {
			next = append(next, a)
		}
	}

	if rep.KitRoot == "" {
		add(Action{
			ID: "kit-root", Severity: SevError,
			Title: "Point kit at the repo",
			Why:   "Nothing else can be compared or captured until the kit root is known.",
			Do:    "export KIT_ROOT=~/dotfiles",
		})
	}

	if rep.Profile == "" {
		add(Action{
			ID: "set-profile", Severity: SevWarn, Runnable: true,
			Title: "Pin this machine as personal or work",
			Why:   "Capture needs a bucket so work plugins do not land on a home laptop.",
			Do:    "kit profile personal    # or: kit profile work",
		})
	}

	var liveSkills, repoOnly, diverge []string
	for _, s := range rep.Skills {
		if s.Ignored {
			continue
		}
		switch s.Presence {
		case PresenceLiveOnly:
			liveSkills = append(liveSkills, s.Name)
		case PresenceRepoOnly:
			repoOnly = append(repoOnly, s.Name)
		}
		if s.Diverges {
			diverge = append(diverge, s.Name)
		}
	}
	if len(liveSkills) > 0 {
		add(Action{
			ID: "capture-skills", Severity: SevWarn, Runnable: true,
			Title: fmt.Sprintf("Save %d skill%s into the kit", len(liveSkills), plural(len(liveSkills))),
			Why:   "They only exist in agent home directories. A new machine will not have them.",
			Do:    "kit capture --skills",
			Items: liveSkills,
		})
	}
	if len(diverge) > 0 {
		add(Action{
			ID: "resolve-skill-drift", Severity: SevWarn,
			Title: "Pick a canonical copy for drifted skills",
			Why:   "The same skill name has different SKILL.md hashes on different agents.",
			Do:    "kit status --json   # inspect hashes, then capture the copy you want",
			Items: diverge,
		})
	}

	var livePlugins, missingInstall []string
	for _, p := range rep.Plugins {
		label := p.Agent + " " + p.ID
		if !p.InManifest && (p.Enabled || p.Installed) {
			livePlugins = append(livePlugins, label)
		}
		if p.Enabled && !p.Installed {
			missingInstall = append(missingInstall, label)
		}
	}
	if len(livePlugins) > 0 {
		add(Action{
			ID: "capture-plugins", Severity: SevWarn, Runnable: true,
			Title: fmt.Sprintf("Record %d plugin%s in the kit manifest", len(livePlugins), plural(len(livePlugins))),
			Why:   "Installing in an agent UI only updates that machine.",
			Do:    "kit capture --plugins",
			Items: livePlugins,
		})
	}
	if len(missingInstall) > 0 {
		add(Action{
			ID: "reinstall-plugins", Severity: SevWarn,
			Title: "Reinstall plugins that are enabled but missing on disk",
			Why:   "Settings ask for them; the install cache does not have them.",
			Do:    "reinstall with the agent's plugin command (kit apply later)",
			Items: missingInstall,
		})
	}

	var settings []string
	for _, s := range rep.Settings {
		if s.Agent == "claude" && fileExists(s.Live) && !s.Linked {
			settings = append(settings, s.Agent+" "+s.File+" is not a kit symlink")
		}
		if s.Drift != "" {
			settings = append(settings, s.Agent+" "+s.File+" drifted from the repo copy")
		}
	}
	if len(settings) > 0 {
		add(Action{
			ID: "capture-settings", Severity: SevWarn, Runnable: true,
			Title: "Harvest live settings into the kit",
			Why:   "The repo copy is stale. The live file is what the tools actually use.",
			Do:    "kit capture --settings",
			Items: settings,
		})
	}

	var locals []string
	for _, f := range rep.Findings {
		switch f.Code {
		case "absolute-path", "marketplace-local-path", "mcp-local-path":
			locals = append(locals, first(f.Item, f.Detail))
		}
	}
	if len(locals) > 0 {
		add(Action{
			ID: "fix-local-paths", Severity: SevWarn,
			Title: "Replace machine-local paths",
			Why:   "These will fail on another computer. Keep them local, or rewrite to ~/ and ${VAR}.",
			Do:    "edit the live config; use ~/… or a command on PATH",
			Items: unique(locals),
		})
	}

	if len(repoOnly) > 0 {
		add(Action{
			ID: "apply-repo-skills", Severity: SevInfo, Later: true,
			Title: "Link kit-only skills onto this machine",
			Why:   "They are in the repo but no agent home has them yet.",
			Do:    "kit apply --skills",
			Items: repoOnly,
		})
	}

	for _, f := range rep.Findings {
		if f.Code == "grok-empty-skills" {
			add(Action{
				ID: "grok-compat", Severity: SevInfo, Later: true,
				Title: "Grok has no ~/.grok/skills (optional)",
				Why:   f.Why,
				Do:    "nothing required — Grok already reads Claude/Cursor skills",
			})
		}
		if f.Code == "git-dirty" {
			add(Action{
				ID: "git-dirty", Severity: SevInfo, Later: true,
				Title: "Kit repo has uncommitted changes",
				Why:   f.Why,
				Do:    "cd " + rep.KitRoot + " && git status",
			})
		}
		if f.Code == "skill-broken-link" {
			add(Action{
				ID: "repair-links", Severity: SevError,
				Title: "Repair broken skill symlinks",
				Why:   f.Detail,
				Do:    "kit apply --skills",
				Items: []string{f.Item},
			})
		}
	}

	rank := 1
	for i := range next {
		next[i].Rank = rank
		rank++
	}
	for i := range later {
		later[i].Rank = rank
		rank++
	}
	return append(next, later...)
}

func FormatDoctor(rep Report, color bool) string {
	return formatActions(rep, color, false)
}

func FormatRecommend(rep Report, color bool) string {
	return formatActions(rep, color, true)
}

func FormatFix(rep Report) string {
	var b strings.Builder
	next, later := SplitActions(rep.Actions)
	if len(next) == 0 {
		fmt.Fprintln(&b, "Nothing to do. Live homes match the kit.")
		return b.String()
	}
	fmt.Fprintln(&b, "Run these, in order. Or: kit doctor  and pick a number.")
	fmt.Fprintln(&b)
	for _, a := range next {
		fmt.Fprintf(&b, "%d.  %s\n", a.Rank, a.Do)
		fmt.Fprintf(&b, "    # %s\n", a.Title)
	}
	if len(later) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Skip unless you care:")
		for _, a := range later {
			fmt.Fprintf(&b, "    %s\n", a.Do)
		}
	}
	return b.String()
}

func formatActions(rep Report, color, showItems bool) string {
	c := palette(color)
	var b strings.Builder
	next, later := SplitActions(rep.Actions)

	fmt.Fprintf(&b, "%skit doctor%s  %s  %s%d error%s  %s%d warn%s  %s%d info%s\n",
		c.bold, c.reset, orDefault(rep.Profile, "no profile"),
		c.red, rep.Summary.Errors, c.reset,
		c.amber, rep.Summary.Warnings, c.reset,
		c.dim, rep.Summary.Infos, c.reset,
	)
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "%sDo this next%s\n", c.bold, c.reset)
	if len(next) == 0 {
		fmt.Fprintf(&b, "  nothing. live homes match the kit.\n")
	}
	for _, a := range next {
		fmt.Fprintf(&b, "  %d. %s%s%s\n", a.Rank, c.bold, a.Title, c.reset)
		fmt.Fprintf(&b, "     %s\n", a.Why)
		fmt.Fprintf(&b, "     %s%s%s\n", c.cyan, a.Do, c.reset)
		if showItems && len(a.Items) > 0 {
			fmt.Fprintf(&b, "     %s%s%s\n", c.dim, joinLimited(a.Items, 8), c.reset)
		}
	}

	if len(later) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "%sLater / ignore%s\n", c.bold, c.reset)
		for _, a := range later {
			fmt.Fprintf(&b, "  · %s\n", a.Title)
			fmt.Fprintf(&b, "    %s%s%s\n", c.dim, a.Why, c.reset)
		}
	}
	if n := CountRunnable(next); n > 0 {
		fmt.Fprintf(&b, "\n%s%s%s\n", c.dim, FormatPrompt(next), c.reset)
	} else {
		fmt.Fprintf(&b, "\n%sFull inventory:%s  kit status     %sVisual:%s  kit board\n", c.dim, c.reset, c.dim, c.reset)
	}
	return b.String()
}

func FormatPrompt(next []Action) string {
	var keys []string
	for _, a := range next {
		if a.Runnable {
			keys = append(keys, fmt.Sprintf("%d", a.Rank))
		}
	}
	if len(keys) == 0 {
		return ""
	}
	return "[" + strings.Join(keys, "] [") + "] run that step    a=all runnable    q=quit"
}

func CountRunnable(acts []Action) int {
	n := 0
	for _, a := range acts {
		if a.Runnable {
			n++
		}
	}
	return n
}

func SplitActions(acts []Action) (next, later []Action) {
	for _, a := range acts {
		if a.Later {
			later = append(later, a)
		} else {
			next = append(next, a)
		}
	}
	return next, later
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func unique(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func joinLimited(ss []string, n int) string {
	if len(ss) <= n {
		return strings.Join(ss, ", ")
	}
	return strings.Join(ss[:n], ", ") + fmt.Sprintf("  +%d more", len(ss)-n)
}
