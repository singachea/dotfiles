package kit

import "time"

type Presence string

const (
	PresenceLiveOnly Presence = "live-only"
	PresenceRepoOnly Presence = "repo-only"
	PresenceBoth     Presence = "both"
)

type CopyKind string

const (
	CopyMissing    CopyKind = "missing"
	CopyDirectory  CopyKind = "directory"
	CopySymlink    CopyKind = "symlink"
	CopyBrokenLink CopyKind = "broken-symlink"
	CopyFile       CopyKind = "file"
)

type Severity string

const (
	SevError Severity = "error"
	SevWarn  Severity = "warn"
	SevInfo  Severity = "info"
)

type Options struct {
	Home    string
	Root    string
	Profile string
	PATH    string
}

type Report struct {
	GeneratedAt  time.Time     `json:"generated_at"`
	Home         string        `json:"home"`
	KitRoot      string        `json:"kit_root"`
	Profile      string        `json:"profile"`
	Git          GitState      `json:"git"`
	Agents       []Agent       `json:"agents"`
	Skills       []Skill       `json:"skills"`
	Plugins      []Plugin      `json:"plugins"`
	Marketplaces []Marketplace `json:"marketplaces"`
	MCP          []MCP         `json:"mcp"`
	Settings     []Setting     `json:"settings"`
	Findings     []Finding     `json:"findings"`
	Actions      []Action      `json:"actions"`
	Summary      Summary       `json:"summary"`
}

type Action struct {
	Rank     int      `json:"rank"`
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Why      string   `json:"why"`
	Do       string   `json:"do"`
	Count    int      `json:"count"`
	Items    []string `json:"items,omitempty"`
	Severity Severity `json:"severity"`
	Later    bool     `json:"later"`
	Runnable bool     `json:"runnable,omitempty"`
}

type Summary struct {
	AgentsPresent int `json:"agents_present"`
	Skills        int `json:"skills"`
	Plugins       int `json:"plugins"`
	Findings      int `json:"findings"`
	Errors        int `json:"errors"`
	Warnings      int `json:"warnings"`
	Infos         int `json:"infos"`
}

type GitState struct {
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Head   string `json:"head,omitempty"`
	Dirty  bool   `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
	Error  string `json:"error,omitempty"`
}

type Agent struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Home    string   `json:"home"`
	Present bool     `json:"present"`
	Binary  string   `json:"binary,omitempty"`
	Notes   []string `json:"notes,omitempty"`
}

type Skill struct {
	Name     string   `json:"name"`
	InRepo   bool     `json:"in_repo"`
	RepoPath string   `json:"repo_path,omitempty"`
	RepoHash string   `json:"repo_hash,omitempty"`
	Profile  string   `json:"profile,omitempty"`
	Presence Presence `json:"presence"`
	Diverges bool     `json:"diverges"`
	Copies   []Copy   `json:"copies"`
	Ignored  bool     `json:"ignored,omitempty"`
	Why      string   `json:"why"`
}

type Copy struct {
	Agent       string   `json:"agent"`
	Path        string   `json:"path"`
	Kind        CopyKind `json:"kind"`
	Target      string   `json:"target,omitempty"`
	Hash        string   `json:"hash,omitempty"`
	MatchesRepo bool     `json:"matches_repo"`
}

type Plugin struct {
	Agent       string `json:"agent"`
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Installed   bool   `json:"installed"`
	Version     string `json:"version,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Marketplace string `json:"marketplace,omitempty"`
	Source      string `json:"source,omitempty"`
	InManifest  bool   `json:"in_manifest"`
	Portable    bool   `json:"portable"`
	Why         string `json:"why"`
}

type Marketplace struct {
	Agent    string `json:"agent"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Source   string `json:"source"`
	Portable bool   `json:"portable"`
	Why      string `json:"why"`
}

type MCP struct {
	Agent    string `json:"agent"`
	Name     string `json:"name"`
	Command  string `json:"command,omitempty"`
	URL      string `json:"url,omitempty"`
	Portable bool   `json:"portable"`
	Why      string `json:"why"`
}

type Setting struct {
	Agent  string `json:"agent"`
	File   string `json:"file"`
	Live   string `json:"live"`
	Repo   string `json:"repo,omitempty"`
	Linked bool   `json:"linked"`
	Drift  string `json:"drift,omitempty"`
	Why    string `json:"why"`
}

type Finding struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Why      string   `json:"why"`
	Fix      string   `json:"fix,omitempty"`
	Agent    string   `json:"agent,omitempty"`
	Item     string   `json:"item,omitempty"`
}

type AgentSpec struct {
	ID         string
	Name       string
	HomeRel    string
	SkillsRel  string
	Binary     string
	GrokCompat bool
}

func AgentSpecs() []AgentSpec {
	return []AgentSpec{
		{ID: "claude", Name: "Claude Code", HomeRel: ".claude", SkillsRel: ".claude/skills", Binary: "claude"},
		{ID: "grok", Name: "Grok", HomeRel: ".grok", SkillsRel: ".grok/skills", Binary: "grok", GrokCompat: true},
		{ID: "codex", Name: "Codex", HomeRel: ".codex", SkillsRel: ".codex/skills", Binary: "codex"},
		{ID: "cursor", Name: "Cursor", HomeRel: ".cursor", SkillsRel: ".cursor/skills", Binary: "cursor"},
		{ID: "opencode", Name: "OpenCode", HomeRel: ".config/opencode", SkillsRel: ".config/opencode/skills", Binary: "opencode"},
		{ID: "kimi", Name: "Kimi", HomeRel: ".agents", SkillsRel: ".agents/skills", Binary: "kimi"},
		{ID: "copilot", Name: "GitHub Copilot", HomeRel: ".copilot", SkillsRel: ".copilot/skills", Binary: "copilot"},
	}
}
