package kit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type CaptureFilter struct {
	Skills   bool
	Plugins  bool
	Settings bool
}

func (f CaptureFilter) any() bool {
	return f.Skills || f.Plugins || f.Settings
}

type CaptureResult struct {
	Skills   int
	Plugins  int
	Settings int
	Wrote    []string
	Skipped  []string
}

func (r CaptureResult) String() string {
	var b []byte
	b = fmt.Appendf(b, "captured  skills=%d  plugins=%d  settings=%d\n", r.Skills, r.Plugins, r.Settings)
	for _, w := range r.Wrote {
		b = fmt.Appendf(b, "  + %s\n", w)
	}
	for _, s := range r.Skipped {
		b = fmt.Appendf(b, "  · %s\n", s)
	}
	return string(b)
}

func Capture(opt Options, filter CaptureFilter) (CaptureResult, error) {
	if !filter.any() {
		filter = CaptureFilter{Skills: true, Plugins: true, Settings: true}
	}
	rep, err := Scan(opt)
	if err != nil {
		return CaptureResult{}, err
	}
	if rep.KitRoot == "" {
		return CaptureResult{}, fmt.Errorf("kit root not found; pass --root or set KIT_ROOT")
	}

	var res CaptureResult
	if filter.Skills {
		if err := captureSkills(rep, &res); err != nil {
			return res, err
		}
	}
	if filter.Plugins {
		if err := capturePlugins(rep, &res); err != nil {
			return res, err
		}
	}
	if filter.Settings {
		if err := captureSettings(rep, &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

func captureSkills(rep Report, res *CaptureResult) error {
	destRoot := filepath.Join(rep.KitRoot, "skills")
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return err
	}
	priority := map[string]int{}
	for i, spec := range AgentSpecs() {
		priority[spec.ID] = i
	}
	for _, s := range rep.Skills {
		if s.Ignored {
			res.Skipped = append(res.Skipped, s.Name+" ignored (.kitignore)")
			continue
		}
		if s.InRepo {
			res.Skipped = append(res.Skipped, s.Name+" already in kit")
			continue
		}
		src := preferredCopy(s, priority)
		if src.Path == "" || src.Kind == CopyMissing || src.Kind == CopyBrokenLink {
			res.Skipped = append(res.Skipped, s.Name+" has no usable live copy")
			continue
		}
		dst := filepath.Join(destRoot, s.Name)
		if err := copyTreeLive(src.Path, dst); err != nil {
			return fmt.Errorf("capture skill %s: %w", s.Name, err)
		}
		res.Skills++
		res.Wrote = append(res.Wrote, "skills/"+s.Name+"  (from "+src.Agent+")")
	}
	return nil
}

func preferredCopy(s Skill, priority map[string]int) Copy {
	best := Copy{}
	bestRank := 1 << 20
	for _, c := range s.Copies {
		if c.Kind == CopyMissing || c.Kind == CopyBrokenLink {
			continue
		}
		r, ok := priority[c.Agent]
		if !ok {
			r = 99
		}
		if r < bestRank {
			best, bestRank = c, r
		}
	}
	return best
}

func capturePlugins(rep Report, res *CaptureResult) error {
	man := loadManifest(rep.KitRoot)
	if man.Plugins == nil {
		man.Plugins = map[string][]string{}
	}
	added := 0
	byAgent := map[string]map[string]bool{}
	for agent, ids := range man.Plugins {
		byAgent[agent] = map[string]bool{}
		for _, id := range ids {
			byAgent[agent][id] = true
		}
	}
	for _, p := range rep.Plugins {
		if !(p.Enabled || p.Installed) {
			continue
		}
		if byAgent[p.Agent] == nil {
			byAgent[p.Agent] = map[string]bool{}
		}
		if byAgent[p.Agent][p.ID] {
			continue
		}
		byAgent[p.Agent][p.ID] = true
		man.Plugins[p.Agent] = append(man.Plugins[p.Agent], p.ID)
		res.Wrote = append(res.Wrote, "manifest.yaml  "+p.Agent+" "+p.ID)
		added++
	}
	for agent, ids := range man.Plugins {
		sort.Strings(ids)
		man.Plugins[agent] = ids
	}
	if added == 0 {
		res.Skipped = append(res.Skipped, "plugins already in manifest")
		return nil
	}
	path := filepath.Join(rep.KitRoot, "manifest.yaml")
	b, err := yamlMarshalManifest(man)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	res.Plugins = added
	return nil
}

func captureSettings(rep Report, res *CaptureResult) error {
	for _, s := range rep.Settings {
		if s.Live == "" || !fileExists(s.Live) {
			continue
		}
		dst := s.Repo
		if dst == "" {
			switch s.Agent {
			case "claude":
				dst = filepath.Join(rep.KitRoot, "claude", s.File)
			default:
				dst = filepath.Join(rep.KitRoot, "hosts", s.Agent, s.File)
			}
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyFile(s.Live, dst); err != nil {
			return fmt.Errorf("capture %s %s: %w", s.Agent, s.File, err)
		}
		res.Settings++
		rel, _ := filepath.Rel(rep.KitRoot, dst)
		res.Wrote = append(res.Wrote, rel)
	}
	return nil
}

func copyTreeLive(src, dst string) error {
	src, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func yamlMarshalManifest(m manifestFile) ([]byte, error) {
	return yaml.Marshal(m)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
