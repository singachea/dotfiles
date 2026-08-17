package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func testOpts(t *testing.T) Options {
	t.Helper()
	home, err := filepath.Abs("testdata/home")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("testdata/kit")
	if err != nil {
		t.Fatal(err)
	}
	return Options{Home: home, Root: root, PATH: t.TempDir()}
}

func TestScanFindsSkillsAcrossAgents(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}

	shared := skillNamed(t, rep, "shared-review")
	if !shared.InRepo {
		t.Fatal("shared-review should be in the kit")
	}
	got := map[string]CopyKind{}
	for _, c := range shared.Copies {
		if c.Kind != CopyMissing {
			got[c.Agent] = c.Kind
		}
	}
	for _, want := range []string{"claude", "codex", "cursor"} {
		if _, ok := got[want]; !ok {
			t.Errorf("shared-review missing on %s", want)
		}
	}
	if got["claude"] != CopyDirectory {
		t.Errorf("claude copy kind = %s, want directory", got["claude"])
	}
}

func TestScanMarksLiveOnlyAndRepoOnlySkills(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}

	only := skillNamed(t, rep, "only-claude")
	if only.InRepo {
		t.Error("only-claude should not be in the kit")
	}
	if only.Presence != PresenceLiveOnly {
		t.Errorf("only-claude presence = %s, want live-only", only.Presence)
	}

	repo := skillNamed(t, rep, "repo-only")
	if !repo.InRepo {
		t.Error("repo-only should be in the kit")
	}
	if repo.Presence != PresenceRepoOnly {
		t.Errorf("repo-only presence = %s, want repo-only", repo.Presence)
	}
}

func TestScanDetectsDivergentSkillHash(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}
	shared := skillNamed(t, rep, "shared-review")
	if !shared.Diverges {
		t.Fatal("cursor copy should make shared-review diverge")
	}
	var cursor Copy
	for _, c := range shared.Copies {
		if c.Agent == "cursor" {
			cursor = c
		}
	}
	if cursor.MatchesRepo {
		t.Error("cursor copy should not match the kit hash")
	}
}

func TestScanReadsClaudeAndGrokPlugins(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}

	sp := pluginNamed(t, rep, "claude", "superpowers@claude-plugins-official")
	if !sp.Enabled || !sp.Installed {
		t.Fatalf("superpowers enabled=%v installed=%v", sp.Enabled, sp.Installed)
	}
	if sp.Version != "6.1.0" {
		t.Errorf("version = %q", sp.Version)
	}
	if sp.InManifest {
		t.Error("live plugin should not be in an empty manifest")
	}

	pt := pluginNamed(t, rep, "grok", "ponytail")
	if !pt.Enabled || !pt.Installed {
		t.Fatalf("grok ponytail enabled=%v installed=%v", pt.Enabled, pt.Installed)
	}
	cf := pluginNamed(t, rep, "grok", "cloudflare")
	if !cf.Enabled {
		t.Fatal("cloudflare should be enabled")
	}
	if cf.Installed {
		t.Error("cloudflare is enabled but not in the registry")
	}
}

func TestScanFlagsNonPortableMarketplaceAndMCP(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}

	var localMkt bool
	for _, m := range rep.Marketplaces {
		if m.Name == "cli-printing-press" && !m.Portable {
			localMkt = true
		}
	}
	if !localMkt {
		t.Fatal("expected non-portable directory marketplace")
	}

	var codedb MCP
	for _, m := range rep.MCP {
		if m.Name == "codedb" {
			codedb = m
		}
	}
	if codedb.Name == "" {
		t.Fatal("missing codedb MCP")
	}
	if codedb.Portable {
		t.Error("absolute MCP command should not be portable")
	}
}

func TestScanEmitsActionableFindings(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"skill-live-only",
		"skill-repo-only",
		"skill-diverges",
		"plugin-live-only",
		"marketplace-local-path",
		"settings-not-linked",
		"settings-drift",
		"absolute-path",
		"mcp-local-path",
		"grok-empty-skills",
	}
	have := map[string]bool{}
	for _, f := range rep.Findings {
		have[f.Code] = true
	}
	for _, code := range want {
		if !have[code] {
			t.Errorf("missing finding %s", code)
		}
	}
	if len(rep.Findings) == 0 {
		t.Fatal("expected findings")
	}
}

func TestTextStatusMentionsFindings(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}
	out := FormatText(rep, false)
	for _, needle := range []string{"FINDINGS", "only-claude", "shared-review", "superpowers", "codedb"} {
		if !hasSubstr(out, needle) {
			t.Errorf("text status missing %q", needle)
		}
	}
}

func TestBoardHTMLContainsMatrix(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderBoard(rep)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"shared-review", "only-claude", "patch-cell", "Do next", "cli-printing-press", "data-tab"} {
		if !hasSubstr(html, needle) {
			t.Errorf("board missing %q", needle)
		}
	}
}

func TestRecommendGroupsRepeatedFindings(t *testing.T) {
	rep, err := Scan(testOpts(t))
	if err != nil {
		t.Fatal(err)
	}
	acts := Recommend(rep)
	if len(acts) == 0 {
		t.Fatal("expected recommended actions")
	}
	ids := map[string]Action{}
	for _, a := range acts {
		ids[a.ID] = a
	}
	if _, ok := ids["capture-skills"]; !ok {
		t.Fatal("expected one capture-skills action, not a row per skill")
	}
	if ids["capture-skills"].Count < 1 {
		t.Fatal("capture-skills should name the live-only skills")
	}
	if _, ok := ids["capture-plugins"]; !ok {
		t.Fatal("expected grouped plugin harvest")
	}
	out := FormatDoctor(rep, false)
	if hasSubstr(out, "SKILLS  (columns") {
		t.Fatal("doctor should not dump the full skill matrix")
	}
	if !hasSubstr(out, "Do this next") {
		t.Fatal("doctor should lead with what to do")
	}
}

func TestCaptureSkillsCopiesLiveOnlyIntoKit(t *testing.T) {
	root := t.TempDir()
	srcKit, err := filepath.Abs("testdata/kit")
	if err != nil {
		t.Fatal(err)
	}
	if err := copyTree(srcKit, root); err != nil {
		t.Fatal(err)
	}
	opt := testOpts(t)
	opt.Root = root

	res, err := Capture(opt, CaptureFilter{Skills: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Skills == 0 {
		t.Fatal("expected to capture at least only-claude")
	}
	if !fileExists(filepath.Join(root, "skills", "only-claude", "SKILL.md")) {
		t.Fatal("only-claude was not written into the kit")
	}
	if !fileExists(filepath.Join(root, "skills", "shared-review", "SKILL.md")) {
		t.Fatal("existing kit skill should remain")
	}

	rep, err := Scan(opt)
	if err != nil {
		t.Fatal(err)
	}
	only := skillNamed(t, rep, "only-claude")
	if only.Presence != PresenceBoth {
		t.Fatalf("after capture, only-claude presence = %s", only.Presence)
	}
}

func TestRemoveSkillDropsItFromKitAndStopsRecapture(t *testing.T) {
	root := t.TempDir()
	srcKit, err := filepath.Abs("testdata/kit")
	if err != nil {
		t.Fatal(err)
	}
	if err := copyTree(srcKit, root); err != nil {
		t.Fatal(err)
	}
	opt := testOpts(t)
	opt.Root = root
	if _, err := Capture(opt, CaptureFilter{Skills: true}); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(root, "skills", "only-claude", "SKILL.md")) {
		t.Fatal("precondition: only-claude should be in the kit")
	}

	res, err := Remove(opt, []string{"only-claude"}, RemoveOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) == 0 {
		t.Fatal("expected a removal")
	}
	if fileExists(filepath.Join(root, "skills", "only-claude", "SKILL.md")) {
		t.Fatal("only-claude should be gone from the kit")
	}

	if _, err := Capture(opt, CaptureFilter{Skills: true}); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(root, "skills", "only-claude", "SKILL.md")) {
		t.Fatal("ignored skill should not be recaptured")
	}

	rep, err := Scan(opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range Recommend(rep) {
		if a.ID == "capture-skills" {
			for _, item := range a.Items {
				if item == "only-claude" {
					t.Fatal("doctor should not ask to recapture an ignored skill")
				}
			}
		}
	}
}

func TestCapturePluginsWritesManifest(t *testing.T) {
	root := t.TempDir()
	opt := testOpts(t)
	opt.Root = root
	res, err := Capture(opt, CaptureFilter{Plugins: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Plugins == 0 {
		t.Fatal("expected plugins in the manifest")
	}
	if !fileExists(filepath.Join(root, "manifest.yaml")) {
		t.Fatal("manifest.yaml missing")
	}
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func TestParseChoicePicksRunnableActions(t *testing.T) {
	next := []Action{
		{Rank: 1, ID: "capture-skills", Runnable: true},
		{Rank: 2, ID: "fix-local-paths", Runnable: false},
		{Rank: 3, ID: "capture-plugins", Runnable: true},
	}
	got, quit, err := ParseChoice("1", next)
	if err != nil || quit || len(got) != 1 || got[0].ID != "capture-skills" {
		t.Fatalf("1 => %+v quit=%v err=%v", got, quit, err)
	}
	got, quit, err = ParseChoice("a", next)
	if err != nil || quit || len(got) != 2 {
		t.Fatalf("a => %+v quit=%v err=%v", got, quit, err)
	}
	_, quit, err = ParseChoice("q", next)
	if err != nil || !quit {
		t.Fatalf("q should quit, err=%v", err)
	}
	if _, _, err := ParseChoice("2", next); err == nil {
		t.Fatal("non-runnable step should error")
	}
}

func TestRunActionCapturesSkills(t *testing.T) {
	root := t.TempDir()
	srcKit, err := filepath.Abs("testdata/kit")
	if err != nil {
		t.Fatal(err)
	}
	if err := copyTree(srcKit, root); err != nil {
		t.Fatal(err)
	}
	opt := testOpts(t)
	opt.Root = root
	out, err := RunAction(opt, Action{ID: "capture-skills"})
	if err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(root, "skills", "only-claude", "SKILL.md")) {
		t.Fatalf("run capture-skills wrote nothing: %s", out)
	}
}

func TestProfileRoundTrip(t *testing.T) {
	home := t.TempDir()
	if err := WriteProfile(home, "work"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadProfile(home)
	if err != nil {
		t.Fatal(err)
	}
	if got != "work" {
		t.Fatalf("profile = %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "kit", "profile")); err != nil {
		t.Fatal(err)
	}
}

func skillNamed(t *testing.T, rep Report, name string) Skill {
	t.Helper()
	for _, s := range rep.Skills {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("skill %s not found", name)
	return Skill{}
}

func pluginNamed(t *testing.T, rep Report, agent, id string) Plugin {
	t.Helper()
	for _, p := range rep.Plugins {
		if p.Agent == agent && p.ID == id {
			return p
		}
	}
	t.Fatalf("plugin %s/%s not found", agent, id)
	return Plugin{}
}

func hasSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
