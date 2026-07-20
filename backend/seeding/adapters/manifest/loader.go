package manifest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/goccy/go-yaml"
)

const (
	maxManifestBytes = 1 << 20
	maxIncludeDepth  = 16
	maxManifestFiles = 64
)

func DefaultPath(profile domain.Profile) string {
	return filepath.FromSlash(filepath.Join("seed", "manifests", string(profile)+".yaml"))
}

// LocateDefaultPath は CLI の repository root と Go test の package working directory の
// どちらからでも既定 manifest を見つける。
func LocateDefaultPath(profile domain.Profile) (string, error) {
	relative := DefaultPath(profile)
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(current, relative)
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("default seed manifest %q not found", relative)
		}
		current = parent
	}
}

func Load(path string) (domain.Manifest, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return domain.Manifest{}, fmt.Errorf("resolve seed manifest path: %w", err)
	}
	root, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return domain.Manifest{}, fmt.Errorf("resolve seed manifest root: %w", err)
	}
	state := loadState{root: root, active: map[string]bool{}, loaded: map[string]bool{}}
	manifest, err := state.load(absolute, 0)
	if err != nil {
		return domain.Manifest{}, err
	}
	manifest.Includes = nil
	if err := manifest.Validate(); err != nil {
		return domain.Manifest{}, err
	}
	return manifest, nil
}

type loadState struct {
	root   string
	active map[string]bool
	loaded map[string]bool
	count  int
}

func (s *loadState) load(path string, depth int) (domain.Manifest, error) {
	if depth > maxIncludeDepth {
		return domain.Manifest{}, fmt.Errorf("seed manifest include depth exceeds %d", maxIncludeDepth)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return domain.Manifest{}, fmt.Errorf("resolve seed manifest: %w", err)
	}
	if !contained(s.root, canonical) {
		return domain.Manifest{}, fmt.Errorf("seed manifest include escapes root")
	}
	if s.active[canonical] {
		return domain.Manifest{}, fmt.Errorf("seed manifest include cycle")
	}
	if s.loaded[canonical] {
		return domain.Manifest{}, nil
	}
	s.count++
	if s.count > maxManifestFiles {
		return domain.Manifest{}, fmt.Errorf("seed manifest file count exceeds %d", maxManifestFiles)
	}
	data, err := readBounded(canonical, maxManifestBytes)
	if err != nil {
		return domain.Manifest{}, err
	}
	if hasUnsupportedYAMLFeature(data) {
		return domain.Manifest{}, fmt.Errorf("seed manifest anchors, aliases, and merge keys are not supported")
	}
	var current domain.Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(data), yaml.Strict())
	if err := decoder.Decode(&current); err != nil {
		return domain.Manifest{}, fmt.Errorf("decode seed manifest: %w", err)
	}
	s.active[canonical] = true
	defer delete(s.active, canonical)
	merged := domain.Manifest{SchemaVersion: current.SchemaVersion, Profile: current.Profile}
	for _, include := range current.Includes {
		if filepath.IsAbs(include) || strings.Contains(include, "://") {
			return domain.Manifest{}, fmt.Errorf("seed manifest include must be a local relative path")
		}
		child, err := s.load(filepath.Join(filepath.Dir(canonical), filepath.Clean(include)), depth+1)
		if err != nil {
			return domain.Manifest{}, err
		}
		if child.SchemaVersion != "" && (child.SchemaVersion != current.SchemaVersion || child.Profile != current.Profile) {
			return domain.Manifest{}, fmt.Errorf("included seed manifest schema/profile mismatch")
		}
		merged.Resources = append(merged.Resources, child.Resources...)
		merged.Generators = append(merged.Generators, child.Generators...)
	}
	merged.Resources = append(merged.Resources, current.Resources...)
	merged.Generators = append(merged.Generators, current.Generators...)
	s.loaded[canonical] = true
	return merged, nil
}

func contained(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func readBounded(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open seed manifest: %w", err)
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read seed manifest: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("seed manifest exceeds size limit")
	}
	return data, nil
}

func hasUnsupportedYAMLFeature(data []byte) bool {
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if strings.HasPrefix(line, "<<:") || strings.Contains(line, " &") || strings.HasPrefix(line, "&") ||
			strings.Contains(line, " *") || strings.HasPrefix(line, "*") {
			return true
		}
	}
	return false
}
