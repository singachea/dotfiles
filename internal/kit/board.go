package kit

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed board.html
var boardTpl string

type boardView struct {
	Report
	ProfileLabel string
	AgentCount   int
	GitLine      string
	PresentIDs   []string
	Skills       []boardSkill
	Next         []Action
	Later        []Action
}

type boardSkill struct {
	Skill
	Copies []boardCopy
}

type boardCopy struct {
	Copy
	Class  string
	Mark   string
	Detail string
}

func RenderBoard(rep Report) (string, error) {
	ids := agentIDs(rep)
	show := map[string]bool{}
	for _, id := range ids {
		show[id] = true
	}
	view := boardView{
		Report:       rep,
		ProfileLabel: orDefault(rep.Profile, "no profile"),
		AgentCount:   rep.Summary.AgentsPresent,
		GitLine:      gitLine(rep.Git),
		PresentIDs:   ids,
	}
	view.Next, view.Later = SplitActions(rep.Actions)
	for _, s := range rep.Skills {
		bs := boardSkill{Skill: s}
		for _, c := range s.Copies {
			if !show[c.Agent] {
				continue
			}
			bs.Copies = append(bs.Copies, decorateCopy(c, s))
		}
		view.Skills = append(view.Skills, bs)
	}

	tpl, err := template.New("board").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(boardTpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func decorateCopy(c Copy, s Skill) boardCopy {
	b := boardCopy{Copy: c, Mark: "·", Class: "miss"}
	switch c.Kind {
	case CopyDirectory, CopyFile:
		b.Mark, b.Class = "D", "dir"
		if s.InRepo && !c.MatchesRepo {
			b.Mark, b.Class = "~", "drift"
		}
	case CopySymlink:
		b.Mark, b.Class = "L", "link"
		if s.InRepo && !c.MatchesRepo {
			b.Mark, b.Class = "~", "drift"
		}
	case CopyBrokenLink:
		b.Mark, b.Class = "X", "break"
	}
	b.Detail = fmt.Sprintf("%s / %s\n%s\nkind %s  hash %s", c.Agent, s.Name, c.Path, c.Kind, c.Hash)
	if c.Target != "" {
		b.Detail += "\n→ " + c.Target
	}
	return b
}

func gitLine(g GitState) string {
	if g.Branch == "" {
		return "not a git checkout"
	}
	s := g.Branch + " " + g.Head
	if g.Dirty {
		s += " dirty"
	}
	return s
}

func WriteBoard(rep Report, out string) (string, error) {
	html, err := RenderBoard(rep)
	if err != nil {
		return "", err
	}
	if out == "" {
		out = filepath.Join(os.TempDir(), "kit-board.html")
	}
	if err := os.WriteFile(out, []byte(html), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func OpenBoard(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
