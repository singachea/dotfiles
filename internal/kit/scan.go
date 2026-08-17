package kit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func Scan(opt Options) (Report, error) {
	home := opt.Home
	if home == "" {
		home = os.Getenv("HOME")
	}
	root := opt.Root
	if root == "" {
		root = DetectRoot(home)
	}
	profile := opt.Profile
	if profile == "" {
		p, err := ReadProfile(home)
		if err == nil {
			profile = p
		}
	}

	rep := Report{
		GeneratedAt: time.Now(),
		Home:        home,
		KitRoot:     root,
		Profile:     profile,
		Git:         scanGit(root),
	}

	manifest := loadManifest(root)
	specs := AgentSpecs()
	searchPath := opt.PATH
	if searchPath == "" {
		searchPath = os.Getenv("PATH")
	}

	for _, spec := range specs {
		homeDir := filepath.Join(home, spec.HomeRel)
		ag := Agent{
			ID:      spec.ID,
			Name:    spec.Name,
			Home:    homeDir,
			Present: dirExists(homeDir),
			Binary:  lookPath(spec.Binary, searchPath),
		}
		if spec.GrokCompat && dirExists(homeDir) {
			n := countSkills(filepath.Join(home, spec.SkillsRel))
			if n == 0 {
				ag.Notes = append(ag.Notes, "empty skills dir; Grok still loads Claude/Cursor skills when compat is on (default)")
			}
		}
		rep.Agents = append(rep.Agents, ag)
	}

	rep.Skills = scanSkills(home, root, specs)
	rep.Plugins, rep.Marketplaces = scanPlugins(home, root, manifest)
	rep.MCP = scanMCP(home)
	rep.Settings = scanSettings(home, root)
	rep.Findings = buildFindings(rep)
	rep.Actions = Recommend(rep)
	rep.Summary = summarize(rep)
	return rep, nil
}

func DetectRoot(home string) string {
	if v := os.Getenv("KIT_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err == nil {
		for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			if isKitRoot(dir) {
				return dir
			}
			if dir == filepath.Dir(dir) {
				break
			}
		}
	}
	if home != "" {
		cand := filepath.Join(home, "dotfiles")
		if isKitRoot(cand) {
			return cand
		}
	}
	return ""
}

func isKitRoot(dir string) bool {
	if fileExists(filepath.Join(dir, "cmd", "kit", "main.go")) {
		return true
	}
	if fileExists(filepath.Join(dir, "claude", "settings.json")) && fileExists(filepath.Join(dir, "go.mod")) {
		return true
	}
	return false
}

func scanSkills(home, root string, specs []AgentSpec) []Skill {
	type loc struct {
		agent string
		path  string
	}
	byName := map[string][]loc{}

	if root != "" {
		for _, name := range listSkillNames(filepath.Join(root, "skills")) {
			byName[name] = append(byName[name], loc{agent: "kit", path: filepath.Join(root, "skills", name)})
		}
	}
	for _, spec := range specs {
		dir := filepath.Join(home, spec.SkillsRel)
		for _, name := range listSkillNames(dir) {
			byName[name] = append(byName[name], loc{agent: spec.ID, path: filepath.Join(dir, name)})
		}
	}

	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)

	ignored := loadIgnore(root)
	var out []Skill
	for _, name := range names {
		s := Skill{Name: name, Profile: "shared", Ignored: ignored[name]}
		repoDir := filepath.Join(root, "skills", name)
		if root != "" && dirExists(repoDir) {
			s.InRepo = true
			s.RepoPath = repoDir
			s.RepoHash = skillHash(repoDir)
		}
		liveCount := 0
		hashes := map[string]int{}
		for _, spec := range specs {
			c := inspectCopy(spec.ID, filepath.Join(home, spec.SkillsRel, name), s.RepoHash)
			if c.Kind != CopyMissing {
				liveCount++
				if c.Hash != "" {
					hashes[c.Hash]++
				}
			}
			s.Copies = append(s.Copies, c)
		}
		switch {
		case s.InRepo && liveCount == 0:
			s.Presence = PresenceRepoOnly
			s.Why = "This skill is in the kit repo but no agent home has it. Other machines will not see it until you apply, and this machine is not using it."
		case !s.InRepo && liveCount > 0:
			s.Presence = PresenceLiveOnly
			s.Why = "An agent wrote this skill into its home directory. It is not in the kit, so it will not survive a new machine or a kit pull."
		default:
			s.Presence = PresenceBoth
			s.Why = "Present in the kit and on at least one agent."
		}
		if len(hashes) > 1 {
			s.Diverges = true
			s.Why += " Copies do not share the same SKILL.md hash — one agent has drifted."
		}
		if s.Ignored {
			s.Why = "Listed in .kitignore — kit will not harvest or recommend this skill. It may still exist in an agent home."
		}
		out = append(out, s)
	}
	return out
}

func inspectCopy(agent, path, repoHash string) Copy {
	c := Copy{Agent: agent, Path: path, Kind: CopyMissing}
	fi, err := os.Lstat(path)
	if err != nil {
		return c
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		c.Target = target
		if err != nil || !exists(path) {
			c.Kind = CopyBrokenLink
			return c
		}
		c.Kind = CopySymlink
	} else if fi.IsDir() {
		c.Kind = CopyDirectory
	} else {
		c.Kind = CopyFile
	}
	c.Hash = skillHash(path)
	c.MatchesRepo = repoHash != "" && c.Hash == repoHash
	return c
}

func listSkillNames(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if e.Type()&os.ModeSymlink != 0 {
			if fileExists(filepath.Join(p, "SKILL.md")) {
				names = append(names, e.Name())
			}
			continue
		}
		if e.IsDir() && fileExists(filepath.Join(p, "SKILL.md")) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

func countSkills(dir string) int { return len(listSkillNames(dir)) }

func skillHash(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

type manifestFile struct {
	Plugins map[string][]string `yaml:"plugins"`
}

func loadManifest(root string) manifestFile {
	var m manifestFile
	if root == "" {
		return m
	}
	b, err := os.ReadFile(filepath.Join(root, "manifest.yaml"))
	if err != nil {
		return m
	}
	_ = yaml.Unmarshal(b, &m)
	if m.Plugins == nil {
		m.Plugins = map[string][]string{}
	}
	return m
}

func inManifest(m manifestFile, agent, id string) bool {
	for _, p := range m.Plugins[agent] {
		if p == id {
			return true
		}
	}
	return false
}

func scanPlugins(home, root string, man manifestFile) ([]Plugin, []Marketplace) {
	var plugins []Plugin
	var mkts []Marketplace

	// Claude
	settings := readJSONMap(filepath.Join(home, ".claude", "settings.json"))
	enabled := stringBoolMap(nested(settings, "enabledPlugins"))
	extraMkts := nested(settings, "extraKnownMarketplaces")
	installed := readClaudeInstalled(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"))
	known := readJSONMap(filepath.Join(home, ".claude", "plugins", "known_marketplaces.json"))

	seen := map[string]bool{}
	for id := range enabled {
		seen[id] = true
	}
	for id := range installed {
		seen[id] = true
	}
	ids := keys(seen)
	sort.Strings(ids)
	for _, id := range ids {
		inst := installed[id]
		mkt := pluginMarketplace(id)
		src, portable, why := claudeMarketplaceSource(mkt, extraMkts, known)
		p := Plugin{
			Agent:       "claude",
			ID:          id,
			Enabled:     enabled[id],
			Installed:   inst.Installed,
			Version:     inst.Version,
			Scope:       inst.Scope,
			Marketplace: mkt,
			Source:      src,
			InManifest:  inManifest(man, "claude", id),
			Portable:    portable,
			Why:         pluginWhy("claude", id, enabled[id], inst.Installed, inManifest(man, "claude", id), portable, why),
		}
		plugins = append(plugins, p)
	}
	mkts = append(mkts, collectClaudeMarketplaces(extraMkts, known)...)

	// Grok
	gcfg := readGrokConfig(filepath.Join(home, ".grok", "config.toml"))
	greg := parseGrokRegistry(filepath.Join(home, ".grok", "installed-plugins", "registry.json"))
	gseen := map[string]bool{}
	for _, id := range gcfg.Enabled {
		gseen[id] = true
	}
	for id := range greg {
		gseen[id] = true
	}
	gids := keys(gseen)
	sort.Strings(gids)
	for _, id := range gids {
		inst := greg[id]
		p := Plugin{
			Agent:      "grok",
			ID:         id,
			Enabled:    contains(gcfg.Enabled, id),
			Installed:  inst.Installed,
			Source:     inst.URL,
			InManifest: inManifest(man, "grok", id),
			Portable:   inst.URL == "" || !isAbsMachinePath(inst.URL),
			Why:        pluginWhy("grok", id, contains(gcfg.Enabled, id), inst.Installed, inManifest(man, "grok", id), true, ""),
		}
		if inst.URL != "" {
			p.Why += " Source " + inst.URL + "."
		}
		plugins = append(plugins, p)
	}
	for _, src := range gcfg.Marketplaces {
		mkts = append(mkts, Marketplace{
			Agent:    "grok",
			Name:     src.Name,
			Kind:     "git",
			Source:   src.Git,
			Portable: src.Git != "" && !isAbsMachinePath(src.Git),
			Why:      "Declared in ~/.grok/config.toml [[marketplace.sources]].",
		})
	}

	_ = root
	return plugins, mkts
}

type installedInfo struct {
	Installed bool
	Version   string
	Scope     string
}

func readClaudeInstalled(path string) map[string]installedInfo {
	out := map[string]installedInfo{}
	raw := readJSONMap(path)
	plugins, _ := raw["plugins"].(map[string]any)
	for id, v := range plugins {
		info := installedInfo{Installed: true, Scope: "user"}
		arr, _ := v.([]any)
		for _, item := range arr {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			if s, _ := m["scope"].(string); s == "user" || info.Version == "" {
				if s != "" {
					info.Scope = s
				}
				if ver, _ := m["version"].(string); ver != "" {
					info.Version = ver
				}
			}
		}
		out[id] = info
	}
	return out
}

func collectClaudeMarketplaces(extra, known map[string]any) []Marketplace {
	names := map[string]bool{}
	for n := range extra {
		names[n] = true
	}
	for n := range known {
		names[n] = true
	}
	list := keys(names)
	sort.Strings(list)
	var out []Marketplace
	for _, name := range list {
		src, portable, why := claudeMarketplaceSource(name, extra, known)
		kind := "github"
		if strings.Contains(why, "directory") || (!portable && strings.HasPrefix(src, "/")) {
			kind = "directory"
		} else if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "git@") {
			kind = "git"
		}
		out = append(out, Marketplace{
			Agent:    "claude",
			Name:     name,
			Kind:     kind,
			Source:   src,
			Portable: portable,
			Why:      why,
		})
	}
	return out
}

func claudeMarketplaceSource(name string, extra, known map[string]any) (src string, portable bool, why string) {
	portable = true
	why = "GitHub/git marketplace — other machines can add it."
	pick := func(entry any) {
		m, _ := entry.(map[string]any)
		if m == nil {
			return
		}
		s, _ := m["source"].(map[string]any)
		if s == nil {
			return
		}
		kind, _ := s["source"].(string)
		switch kind {
		case "directory":
			src, _ = s["path"].(string)
			portable = false
			why = "Directory marketplace points at a path on this machine. kit will not copy it to another computer."
		case "github":
			src, _ = s["repo"].(string)
		case "git":
			src, _ = s["url"].(string)
		}
	}
	if extra != nil {
		if e, ok := extra[name]; ok {
			pick(e)
		}
	}
	if src == "" && known != nil {
		if e, ok := known[name]; ok {
			pick(e)
		}
	}
	return src, portable, why
}

func pluginMarketplace(id string) string {
	if i := strings.LastIndex(id, "@"); i >= 0 {
		return id[i+1:]
	}
	return ""
}

func pluginWhy(agent, id string, enabled, installed, inMan, portable bool, extra string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s plugin %s: ", agent, id)
	switch {
	case enabled && installed:
		b.WriteString("enabled and installed on this machine.")
	case enabled && !installed:
		b.WriteString("enabled in settings but the install tree is missing.")
	case !enabled && installed:
		b.WriteString("installed on disk but not enabled.")
	default:
		b.WriteString("listed without enabled/installed flags.")
	}
	if !inMan {
		b.WriteString(" Not declared in kit/manifest.yaml, so another machine will not install it.")
	} else {
		b.WriteString(" Declared in the kit manifest.")
	}
	if !portable {
		b.WriteString(" " + extra)
	}
	return b.String()
}

type grokConfig struct {
	Enabled      []string
	Marketplaces []struct {
		Name string
		Git  string
	}
}

func readGrokConfig(path string) grokConfig {
	var out grokConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var raw struct {
		Plugins struct {
			Enabled []string `toml:"enabled"`
		} `toml:"plugins"`
		Marketplace struct {
			Sources []struct {
				Name string `toml:"name"`
				Git  string `toml:"git"`
			} `toml:"sources"`
		} `toml:"marketplace"`
	}
	if err := toml.Unmarshal(b, &raw); err != nil {
		return out
	}
	out.Enabled = raw.Plugins.Enabled
	for _, s := range raw.Marketplace.Sources {
		out.Marketplaces = append(out.Marketplaces, struct {
			Name string
			Git  string
		}{s.Name, s.Git})
	}
	return out
}

type grokInst struct {
	installedInfo
	URL string
}

func parseGrokRegistry(path string) map[string]grokInst {
	out := map[string]grokInst{}
	raw := readJSONMap(path)
	repos, _ := raw["repos"].(map[string]any)
	for _, rv := range repos {
		rm, _ := rv.(map[string]any)
		if rm == nil {
			continue
		}
		url := ""
		if kind, _ := rm["kind"].(map[string]any); kind != nil {
			url, _ = kind["url"].(string)
		}
		plugins, _ := rm["plugins"].(map[string]any)
		for id := range plugins {
			out[id] = grokInst{installedInfo: installedInfo{Installed: true, Scope: "user"}, URL: url}
		}
	}
	return out
}

func scanMCP(home string) []MCP {
	var out []MCP
	// Codex / Grok style TOML
	for _, spec := range []struct{ agent, rel string }{
		{"codex", ".codex/config.toml"},
		{"grok", ".grok/config.toml"},
	} {
		b, err := os.ReadFile(filepath.Join(home, spec.rel))
		if err != nil {
			continue
		}
		var raw map[string]any
		if err := toml.Unmarshal(b, &raw); err != nil {
			continue
		}
		servers, _ := raw["mcp_servers"].(map[string]any)
		names := keys(servers)
		sort.Strings(names)
		for _, name := range names {
			sm, _ := servers[name].(map[string]any)
			cmd, _ := sm["command"].(string)
			url, _ := sm["url"].(string)
			portable := !isAbsMachinePath(cmd) && !isAbsMachinePath(url)
			why := "MCP server " + name + " on " + spec.agent + "."
			if !portable {
				why = "Command/URL is an absolute path on this machine and will break on another computer."
			}
			out = append(out, MCP{Agent: spec.agent, Name: name, Command: cmd, URL: url, Portable: portable, Why: why})
		}
	}
	// Cursor JSON
	cur := readJSONMap(filepath.Join(home, ".cursor", "mcp.json"))
	if servers, ok := cur["mcpServers"].(map[string]any); ok {
		names := keys(servers)
		sort.Strings(names)
		for _, name := range names {
			sm, _ := servers[name].(map[string]any)
			cmd, _ := sm["command"].(string)
			url, _ := sm["url"].(string)
			portable := !isAbsMachinePath(cmd)
			out = append(out, MCP{Agent: "cursor", Name: name, Command: cmd, URL: url, Portable: portable, Why: "From ~/.cursor/mcp.json."})
		}
	}
	return out
}

func scanSettings(home, root string) []Setting {
	var out []Setting
	live := filepath.Join(home, ".claude", "settings.json")
	repo := ""
	if root != "" {
		if fileExists(filepath.Join(root, "hosts", "claude", "settings.json")) {
			repo = filepath.Join(root, "hosts", "claude", "settings.json")
		} else if fileExists(filepath.Join(root, "claude", "settings.json")) {
			repo = filepath.Join(root, "claude", "settings.json")
		}
	}
	s := Setting{
		Agent: "claude",
		File:  "settings.json",
		Live:  live,
		Repo:  repo,
		Why:   "Claude writes plugins, hooks, and model prefs here. If this file is not a symlink into the kit, the repo copy will drift.",
	}
	if fileExists(live) {
		if target, err := os.Readlink(live); err == nil {
			s.Linked = true
			s.Why = "Live settings.json is a symlink → " + target
			if repo != "" {
				absRepo, _ := filepath.Abs(repo)
				absT, _ := filepath.Abs(filepath.Join(filepath.Dir(live), target))
				if absT != absRepo && target != repo {
					s.Drift = "symlink does not point at the kit copy"
				}
			}
		} else if repo != "" {
			s.Linked = false
			ld := claudeEnabledList(live)
			rd := claudeEnabledList(repo)
			if strings.Join(ld, ",") != strings.Join(rd, ",") {
				s.Drift = fmt.Sprintf("enabled plugins live=%v kit=%v", ld, rd)
			}
		}
	}
	out = append(out, s)

	// grok config
	glive := filepath.Join(home, ".grok", "config.toml")
	if fileExists(glive) {
		gs := Setting{Agent: "grok", File: "config.toml", Live: glive, Why: "Grok user config (plugins, model, UI). Not in the kit yet."}
		if root != "" && fileExists(filepath.Join(root, "hosts", "grok", "config.toml")) {
			gs.Repo = filepath.Join(root, "hosts", "grok", "config.toml")
		}
		if _, err := os.Readlink(glive); err == nil {
			gs.Linked = true
		}
		out = append(out, gs)
	}
	return out
}

func claudeEnabledList(path string) []string {
	m := stringBoolMap(nested(readJSONMap(path), "enabledPlugins"))
	ks := keys(m)
	sort.Strings(ks)
	return ks
}

func scanGit(root string) GitState {
	g := GitState{Repo: root}
	if root == "" {
		g.Error = "kit root not found"
		return g
	}
	if _, err := exec.LookPath("git"); err != nil {
		g.Error = "git not on PATH"
		return g
	}
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		b, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	g.Branch = run("rev-parse", "--abbrev-ref", "HEAD")
	g.Head = run("rev-parse", "--short", "HEAD")
	if run("status", "--porcelain") != "" {
		g.Dirty = true
	}
	ab := run("rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if ab != "" {
		var behind, ahead int
		fmt.Sscanf(ab, "%d\t%d", &behind, &ahead)
		g.Behind, g.Ahead = behind, ahead
	}
	return g
}

func lookPath(name, search string) string {
	if name == "" {
		return ""
	}
	old := os.Getenv("PATH")
	if search != "" {
		_ = os.Setenv("PATH", search)
		defer os.Setenv("PATH", old)
	}
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func readJSONMap(path string) map[string]any {
	b, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func nested(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

func stringBoolMap(m map[string]any) map[string]bool {
	out := map[string]bool{}
	for k, v := range m {
		switch t := v.(type) {
		case bool:
			out[k] = t
		default:
			out[k] = true
		}
	}
	return out
}

func keys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func isAbsMachinePath(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "~/") {
		return false
	}
	if strings.HasPrefix(s, "/Users/") || strings.HasPrefix(s, "/home/") {
		return true
	}
	return filepath.IsAbs(s) && !strings.HasPrefix(s, "/usr/") && !strings.HasPrefix(s, "/opt/") && !strings.HasPrefix(s, "/bin")
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
