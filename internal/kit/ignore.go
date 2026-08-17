package kit

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ignorePath(root string) string {
	return filepath.Join(root, ".kitignore")
}

func loadIgnore(root string) map[string]bool {
	out := map[string]bool{}
	if root == "" {
		return out
	}
	f, err := os.Open(ignorePath(root))
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "skill:")
		out[line] = true
	}
	return out
}

func addIgnore(root string, names []string) error {
	have := loadIgnore(root)
	for _, n := range names {
		have[n] = true
	}
	keys := make([]string, 0, len(have))
	for k := range have {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# skills kit should not harvest or recommend\n")
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('\n')
	}
	return os.WriteFile(ignorePath(root), []byte(b.String()), 0o644)
}
