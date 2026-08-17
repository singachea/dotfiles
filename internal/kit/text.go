package kit

import (
	"fmt"
	"strings"
)

func FormatText(rep Report, color bool) string {
	c := palette(color)
	var b strings.Builder

	profile := rep.Profile
	if profile == "" {
		profile = "profile-unset"
	}
	fmt.Fprintf(&b, "%skit%s  %s  %s\n", c.bold, c.reset, profile, dash(rep.KitRoot, "(no kit root)"))
	if rep.Git.Branch != "" {
		dirty := ""
		if rep.Git.Dirty {
			dirty = " dirty"
		}
		fmt.Fprintf(&b, "     %s %s%s  ahead %d  behind %d\n", rep.Git.Branch, rep.Git.Head, dirty, rep.Git.Ahead, rep.Git.Behind)
	}
	fmt.Fprintf(&b, "     %d agents  %d skills  %d plugins  %s%d error%s  %s%d warn%s  %s%d info%s\n\n",
		rep.Summary.AgentsPresent, rep.Summary.Skills, rep.Summary.Plugins,
		c.red, rep.Summary.Errors, c.reset,
		c.amber, rep.Summary.Warnings, c.reset,
		c.dim, rep.Summary.Infos, c.reset,
	)

	fmt.Fprintf(&b, "%sFINDINGS%s  (%d) — why each thing is happening\n", c.bold, c.reset, len(rep.Findings))
	if len(rep.Findings) == 0 {
		fmt.Fprintf(&b, "  %snothing to report%s\n", c.green, c.reset)
	}
	for _, f := range rep.Findings {
		mark := "?"
		col := c.dim
		switch f.Severity {
		case SevError:
			mark, col = "x", c.red
		case SevWarn:
			mark, col = "!", c.amber
		case SevInfo:
			mark, col = "i", c.cyan
		}
		fmt.Fprintf(&b, "  %s%s%s  %s%s%s\n", col, mark, c.reset, c.bold, f.Title, c.reset)
		fmt.Fprintf(&b, "      %swhy%s  %s\n", c.dim, c.reset, f.Why)
		if f.Detail != "" {
			fmt.Fprintf(&b, "      %snow%s  %s\n", c.dim, c.reset, f.Detail)
		}
		if f.Fix != "" {
			fmt.Fprintf(&b, "      %sfix%s  %s\n", c.dim, c.reset, f.Fix)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "%sAGENTS%s\n", c.bold, c.reset)
	for _, a := range rep.Agents {
		state := "absent"
		col := c.dim
		if a.Present {
			state, col = "home", c.green
		}
		bin := "no binary"
		if a.Binary != "" {
			bin = a.Binary
		}
		fmt.Fprintf(&b, "  %-8s  %s%-6s%s  %-40s  %s\n", a.ID, col, state, c.reset, trunc(a.Home, 40), bin)
		for _, n := range a.Notes {
			fmt.Fprintf(&b, "           %s%s%s\n", c.dim, n, c.reset)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "%sSKILLS%s  (columns = agent homes; L=symlink  D=copy  ~=hash mismatch  .=missing)\n", c.bold, c.reset)
	ids := agentIDs(rep)
	fmt.Fprintf(&b, "  %-24s %-6s", "name", "kit")
	for _, id := range ids {
		fmt.Fprintf(&b, " %-7s", id)
	}
	fmt.Fprintln(&b)
	for _, s := range rep.Skills {
		kit := "."
		if s.InRepo {
			kit = "ok"
		}
		fmt.Fprintf(&b, "  %-24s %-6s", s.Name, kit)
		by := map[string]Copy{}
		for _, cp := range s.Copies {
			by[cp.Agent] = cp
		}
		for _, id := range ids {
			fmt.Fprintf(&b, " %-7s", cell(by[id], s))
		}
		fmt.Fprintln(&b)
		if s.Diverges || s.Presence == PresenceLiveOnly {
			fmt.Fprintf(&b, "    %s%s%s\n", c.dim, s.Why, c.reset)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "%sPLUGINS%s\n", c.bold, c.reset)
	for _, p := range rep.Plugins {
		flags := []string{}
		if p.Enabled {
			flags = append(flags, "enabled")
		} else {
			flags = append(flags, "disabled")
		}
		if p.Installed {
			flags = append(flags, "installed")
		} else {
			flags = append(flags, "not-installed")
		}
		if p.InManifest {
			flags = append(flags, "in-kit")
		} else {
			flags = append(flags, "not-in-kit")
		}
		if !p.Portable {
			flags = append(flags, "not-portable")
		}
		fmt.Fprintf(&b, "  %-8s %-42s %s", p.Agent, p.ID, strings.Join(flags, "  "))
		if p.Version != "" {
			fmt.Fprintf(&b, "  %s", p.Version)
		}
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "           %s%s%s\n", c.dim, p.Why, c.reset)
	}
	fmt.Fprintln(&b)

	if len(rep.Marketplaces) > 0 {
		fmt.Fprintf(&b, "%sMARKETPLACES%s\n", c.bold, c.reset)
		for _, m := range rep.Marketplaces {
			port := "portable"
			if !m.Portable {
				port = "LOCAL"
			}
			fmt.Fprintf(&b, "  %-8s %-22s %-10s %-8s %s\n", m.Agent, m.Name, m.Kind, port, m.Source)
			fmt.Fprintf(&b, "           %s%s%s\n", c.dim, m.Why, c.reset)
		}
		fmt.Fprintln(&b)
	}

	if len(rep.MCP) > 0 {
		fmt.Fprintf(&b, "%sMCP%s\n", c.bold, c.reset)
		for _, m := range rep.MCP {
			port := "portable"
			if !m.Portable {
				port = "LOCAL"
			}
			fmt.Fprintf(&b, "  %-8s %-16s %-8s %s\n", m.Agent, m.Name, port, first(m.Command, m.URL))
			fmt.Fprintf(&b, "           %s%s%s\n", c.dim, m.Why, c.reset)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "%sSETTINGS%s\n", c.bold, c.reset)
	for _, s := range rep.Settings {
		link := "file"
		if s.Linked {
			link = "symlink"
		}
		fmt.Fprintf(&b, "  %-8s %-16s %-8s %s\n", s.Agent, s.File, link, s.Live)
		if s.Repo != "" {
			fmt.Fprintf(&b, "           kit copy  %s\n", s.Repo)
		}
		if s.Drift != "" {
			fmt.Fprintf(&b, "           %sdrift%s  %s\n", c.amber, c.reset, s.Drift)
		}
		fmt.Fprintf(&b, "           %s%s%s\n", c.dim, s.Why, c.reset)
	}

	fmt.Fprintf(&b, "\n%sOpen the board for the skill×agent patch grid and the same explanations in a page:%s\n  kit board\n", c.dim, c.reset)
	return b.String()
}

func agentIDs(rep Report) []string {
	var ids []string
	for _, a := range rep.Agents {
		if a.Present {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		for _, a := range rep.Agents {
			ids = append(ids, a.ID)
		}
	}
	return ids
}

func cell(c Copy, s Skill) string {
	switch c.Kind {
	case CopyMissing:
		return "."
	case CopyBrokenLink:
		return "X"
	case CopySymlink:
		if s.InRepo && !c.MatchesRepo {
			return "L~"
		}
		return "L"
	default:
		if s.InRepo && !c.MatchesRepo {
			return "D~"
		}
		return "D"
	}
}

func dash(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

type colors struct{ bold, dim, red, amber, green, cyan, reset string }

func palette(on bool) colors {
	if !on {
		return colors{}
	}
	return colors{
		bold:  "\033[1m",
		dim:   "\033[2m",
		red:   "\033[31m",
		amber: "\033[33m",
		green: "\033[32m",
		cyan:  "\033[36m",
		reset: "\033[0m",
	}
}
