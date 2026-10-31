// Package cli は、設定ファイルを読み、ゲートウェイを組み立てて待ち受ける。
package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

const (
	defaultMaxBodyBytes = 4 << 20
	defaultTimeout      = 5 * time.Second
)

type fileOperation struct {
	ResourceType       string `yaml:"resource_type"`
	Relation           string `yaml:"relation"`
	ResourceID         string `yaml:"resource_id"`
	ResourceIDArgument string `yaml:"resource_id_argument"`
}

// fileConfig は設定ファイルの形である。未知のキーは読み取りで拒否する (REQ-MCPGATEWAY-001)。
type fileConfig struct {
	Listen        string `yaml:"listen"`
	MetricsListen string `yaml:"metrics_listen"`
	Resource      string `yaml:"resource"`
	Upstream      string `yaml:"upstream"`
	IdMagic       struct {
		RealmURL         string `yaml:"realm_url"`
		CAFile           string `yaml:"ca_file"`
		ClientID         string `yaml:"client_id"`
		ClientSecretFile string `yaml:"client_secret_file"`
		Timeout          string `yaml:"timeout"`
	} `yaml:"idmagic"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	MaxBodyBytes   *int64   `yaml:"max_body_bytes"`
	Mapping        struct {
		Methods map[string]fileOperation `yaml:"methods"`
		Tools   map[string]fileOperation `yaml:"tools"`
	} `yaml:"mapping"`
}

// Config は検証済みの設定である。
type Config struct {
	Listen         string
	MetricsListen  string
	Resource       *url.URL
	Upstream       *url.URL
	RealmURL       string
	TLS            *tls.Config
	ClientID       string
	ClientSecret   string
	Timeout        time.Duration
	AllowedOrigins []string
	MaxBodyBytes   int64
	Mapping        domain.Mapping
}

// LoadConfig は設定ファイルを読み、どの誤りも起動前に拒否する。
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var file fileConfig
	if err := yaml.UnmarshalWithOptions(raw, &file, yaml.DisallowUnknownField()); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	config := Config{
		Listen: file.Listen, MetricsListen: file.MetricsListen, ClientID: file.IdMagic.ClientID,
		AllowedOrigins: file.AllowedOrigins, MaxBodyBytes: defaultMaxBodyBytes, Timeout: defaultTimeout,
	}
	if config.Listen == "" {
		return Config{}, errors.New("listen is required")
	}
	if config.ClientID == "" {
		return Config{}, errors.New("idmagic.client_id is required")
	}
	if config.Resource, err = absoluteURL("resource", file.Resource, "https"); err != nil {
		return Config{}, err
	}
	if config.Upstream, err = absoluteURL("upstream", file.Upstream, "http", "https"); err != nil {
		return Config{}, err
	}
	realm, err := absoluteURL("idmagic.realm_url", file.IdMagic.RealmURL, "https")
	if err != nil {
		return Config{}, err
	}
	config.RealmURL = strings.TrimSuffix(realm.String(), "/")
	if file.MaxBodyBytes != nil {
		if *file.MaxBodyBytes <= 0 {
			return Config{}, errors.New("max_body_bytes must be positive")
		}
		config.MaxBodyBytes = *file.MaxBodyBytes
	}
	if file.IdMagic.Timeout != "" {
		if config.Timeout, err = time.ParseDuration(file.IdMagic.Timeout); err != nil || config.Timeout <= 0 {
			return Config{}, errors.New("idmagic.timeout must be a positive duration")
		}
	}
	if config.ClientSecret, err = readSecret(file.IdMagic.ClientSecretFile); err != nil {
		return Config{}, err
	}
	if config.TLS, err = tlsConfig(file.IdMagic.CAFile); err != nil {
		return Config{}, err
	}
	if config.Mapping, err = mapping(file); err != nil {
		return Config{}, err
	}
	return config, nil
}

func absoluteURL(name, value string, schemes ...string) (*url.URL, error) {
	if value == "" {
		return nil, fmt.Errorf("%s is required", name)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return nil, fmt.Errorf("%s must be an absolute URL without credentials, query or fragment", name)
	}
	if slices.Contains(schemes, parsed.Scheme) {
		return parsed, nil
	}
	return nil, fmt.Errorf("%s must use %s", name, strings.Join(schemes, " or "))
}

func readSecret(path string) (string, error) {
	if path == "" {
		return "", errors.New("idmagic.client_secret_file is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("idmagic.client_secret_file could not be read")
	}
	secret := strings.TrimRight(string(raw), "\r\n")
	if secret == "" {
		return "", errors.New("idmagic.client_secret_file is empty")
	}
	return secret, nil
}

func tlsConfig(caFile string) (*tls.Config, error) {
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile == "" {
		return config, nil
	}
	raw, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read idmagic.ca_file: %w", err)
	}
	config.RootCAs = x509.NewCertPool()
	if !config.RootCAs.AppendCertsFromPEM(raw) {
		return nil, errors.New("idmagic.ca_file has no certificate")
	}
	return config, nil
}

func mapping(file fileConfig) (domain.Mapping, error) {
	result := domain.Mapping{Methods: map[string]domain.OperationRule{}, Tools: map[string]domain.OperationRule{}}
	for method, operation := range file.Mapping.Methods {
		if method == domain.MethodToolsCall {
			return domain.Mapping{}, errors.New("mapping.methods must not name tools/call; declare mapping.tools")
		}
		if operation.ResourceIDArgument != "" {
			return domain.Mapping{}, fmt.Errorf("mapping.methods.%s must use resource_id", method)
		}
		rule, err := operationRule("mapping.methods."+method, operation)
		if err != nil {
			return domain.Mapping{}, err
		}
		result.Methods[method] = rule
	}
	for tool, operation := range file.Mapping.Tools {
		rule, err := operationRule("mapping.tools."+tool, operation)
		if err != nil {
			return domain.Mapping{}, err
		}
		result.Tools[tool] = rule
	}
	return result, nil
}

func operationRule(name string, operation fileOperation) (domain.OperationRule, error) {
	if operation.ResourceType == "" || operation.Relation == "" {
		return domain.OperationRule{}, fmt.Errorf("%s requires resource_type and relation", name)
	}
	if (operation.ResourceID == "") == (operation.ResourceIDArgument == "") {
		return domain.OperationRule{}, fmt.Errorf("%s requires exactly one of resource_id and resource_id_argument", name)
	}
	return domain.OperationRule{
		ResourceType: operation.ResourceType, Relation: operation.Relation,
		ResourceID: operation.ResourceID, ResourceIDArgument: operation.ResourceIDArgument,
	}, nil
}
