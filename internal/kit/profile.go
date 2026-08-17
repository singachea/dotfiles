package kit

import (
	"os"
	"path/filepath"
	"strings"
)

func profilePath(home string) string {
	return filepath.Join(home, ".config", "kit", "profile")
}

func ReadProfile(home string) (string, error) {
	if v := os.Getenv("KIT_PROFILE"); v != "" {
		return strings.TrimSpace(v), nil
	}
	b, err := os.ReadFile(profilePath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func WriteProfile(home, name string) error {
	name = strings.TrimSpace(name)
	if name != "personal" && name != "work" {
		return os.ErrInvalid
	}
	dir := filepath.Dir(profilePath(home))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(profilePath(home), []byte(name+"\n"), 0o644)
}
