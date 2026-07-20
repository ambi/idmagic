package manifests_yaml

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ambi/idmagic/backend/seeding/domain"
)

const MaximumSecretBytes = 64 * 1024

type SecretResolver struct {
	SecretRoot string
	Getenv     func(string) string
}

func (r SecretResolver) Resolve(reference domain.SecretReference) (string, error) {
	if err := reference.Validate(); err != nil {
		return "", err
	}
	switch reference.Provider {
	case domain.SecretProviderEnv:
		if r.Getenv == nil {
			return "", fmt.Errorf("seed env secret provider is not configured")
		}
		value := r.Getenv(reference.Locator)
		if value == "" {
			return "", fmt.Errorf("seed env secret is unavailable")
		}
		if strings.IndexByte(value, 0) >= 0 {
			return "", fmt.Errorf("seed env secret contains NUL")
		}
		return value, nil
	case domain.SecretProviderFile:
		return r.resolveFile(reference.Locator)
	default:
		return "", fmt.Errorf("unsupported seed secret provider")
	}
}

func (r SecretResolver) resolveFile(locator string) (string, error) {
	if r.SecretRoot == "" || filepath.IsAbs(locator) {
		return "", fmt.Errorf("seed secret file root or locator is invalid")
	}
	root, err := filepath.EvalSymlinks(r.SecretRoot)
	if err != nil {
		return "", fmt.Errorf("resolve seed secret root: %w", err)
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, filepath.Clean(locator)))
	if err != nil {
		return "", fmt.Errorf("resolve seed secret file: %w", err)
	}
	if !contained(root, path) {
		return "", fmt.Errorf("seed secret file escapes root")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open seed secret file: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("seed secret file must be regular")
	}
	data, err := io.ReadAll(io.LimitReader(file, MaximumSecretBytes+1))
	if err != nil {
		return "", fmt.Errorf("read seed secret file: %w", err)
	}
	if len(data) > MaximumSecretBytes {
		return "", fmt.Errorf("seed secret file exceeds size limit")
	}
	if len(data) == 0 || bytes.IndexByte(data, 0) >= 0 {
		return "", fmt.Errorf("seed secret file is empty or contains NUL")
	}
	data = bytes.TrimSuffix(data, []byte("\r\n"))
	data = bytes.TrimSuffix(data, []byte("\n"))
	if len(data) == 0 {
		return "", fmt.Errorf("seed secret file is empty")
	}
	return string(data), nil
}
