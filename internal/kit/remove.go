package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RemoveOpts struct {
	NoIgnore bool
}

type RemoveResult struct {
	Removed []string
	Ignored []string
	Missing []string
}

func (r RemoveResult) String() string {
	var b strings.Builder
	if len(r.Removed) == 0 && len(r.Missing) == 0 {
		fmt.Fprintln(&b, "nothing to remove")
	}
	for _, n := range r.Removed {
		fmt.Fprintf(&b, "removed  skills/%s\n", n)
	}
	for _, n := range r.Ignored {
		fmt.Fprintf(&b, "ignored  %s  (capture/doctor will skip it)\n", n)
	}
	for _, n := range r.Missing {
		fmt.Fprintf(&b, "missing  skills/%s\n", n)
	}
	return b.String()
}

func Remove(opt Options, names []string, ropt RemoveOpts) (RemoveResult, error) {
	var res RemoveResult
	if len(names) == 0 {
		return res, fmt.Errorf("usage: kit remove <skill> [skill...]")
	}
	root := opt.Root
	if root == "" {
		home := opt.Home
		if home == "" {
			home = os.Getenv("HOME")
		}
		root = DetectRoot(home)
	}
	if root == "" {
		return res, fmt.Errorf("kit root not found; pass --root or set KIT_ROOT")
	}

	var toIgnore []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		name = strings.TrimSuffix(name, "/")
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if name == "" {
			continue
		}
		dir := filepath.Join(root, "skills", name)
		if dirExists(dir) {
			if err := os.RemoveAll(dir); err != nil {
				return res, fmt.Errorf("remove %s: %w", name, err)
			}
			res.Removed = append(res.Removed, name)
		} else {
			res.Missing = append(res.Missing, name)
		}
		toIgnore = append(toIgnore, name)
	}
	if !ropt.NoIgnore && len(toIgnore) > 0 {
		if err := addIgnore(root, toIgnore); err != nil {
			return res, err
		}
		res.Ignored = toIgnore
	}
	return res, nil
}
