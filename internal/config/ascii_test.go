package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayASCIIDefaultAndOverride(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		want        bool
	}{
		{"default", "", false},
		{"enabled", "ascii = true\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			data := "[general]\nusername='me'\n[[organizations]]\nlogin='org'\n[display]\n" + tc.extra
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadFromPath(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Display.ASCII != tc.want {
				t.Fatalf("ASCII=%v want %v", cfg.Display.ASCII, tc.want)
			}
		})
	}
}
