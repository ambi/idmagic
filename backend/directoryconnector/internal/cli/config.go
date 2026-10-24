// Package cli は、Connector の設定ファイルを読み、登録と同期のサブコマンドを組み立てる。
package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ldap_goldap"
)

// fileConfig は設定ファイルの形である。未知のキーは読み取りで拒否する (REQ-DIRECTORYCONNECTOR-002)。
type fileConfig struct {
	IdMagic struct {
		RealmURL string `yaml:"realm_url"`
		CAFile   string `yaml:"ca_file"`
		ProxyURL string `yaml:"proxy_url"`
	} `yaml:"idmagic"`
	StateDir string `yaml:"state_dir"`
	LDAP     struct {
		URL              string            `yaml:"url"`
		StartTLS         bool              `yaml:"start_tls"`
		CAFile           string            `yaml:"ca_file"`
		ServerName       string            `yaml:"server_name"`
		Dialect          string            `yaml:"dialect"`
		BindDN           string            `yaml:"bind_dn"`
		BindPasswordFile string            `yaml:"bind_password_file"`
		BaseDNs          []string          `yaml:"base_dns"`
		UserFilter       string            `yaml:"user_filter"`
		PageSize         *int              `yaml:"page_size"`
		Attributes       map[string]string `yaml:"attributes"`
	} `yaml:"ldap"`
	Sync struct {
		Interval     string `yaml:"interval"`
		FullInterval string `yaml:"full_interval"`
	} `yaml:"sync"`
}

// Config は検証済みの設定である。
type Config struct {
	RealmURL   string
	IdMagicTLS *tls.Config
	// Proxy は IdMagic への要求を中継するプロキシである。nil なら直接つなぐ。
	Proxy        *url.URL
	StateDir     string
	LDAP         ldap_goldap.Config
	Dialect      domain.Dialect
	Scope        domain.Scope
	Mapping      domain.Mapping
	Interval     time.Duration
	FullInterval time.Duration
}

const (
	defaultPageSize     = 500
	maxPageSize         = 1000
	defaultInterval     = 5 * time.Minute
	defaultFullInterval = 24 * time.Hour
)

// LoadConfig は設定ファイルとパスワードのファイルを読み、どこにも接続せずに検証する。
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var file fileConfig
	if err := yaml.UnmarshalWithOptions(raw, &file, yaml.DisallowUnknownField()); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	var problems []string
	require := func(value, name string) {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, name+" is required")
		}
	}
	require(file.IdMagic.RealmURL, "idmagic.realm_url")
	require(file.StateDir, "state_dir")
	require(file.LDAP.URL, "ldap.url")
	require(file.LDAP.Dialect, "ldap.dialect")
	require(file.LDAP.BindDN, "ldap.bind_dn")
	require(file.LDAP.BindPasswordFile, "ldap.bind_password_file")
	if len(problems) > 0 {
		return Config{}, errors.New(strings.Join(problems, "; "))
	}

	config := Config{RealmURL: file.IdMagic.RealmURL, StateDir: file.StateDir}
	if realm, err := url.Parse(file.IdMagic.RealmURL); err != nil || realm.Scheme != "https" || realm.Host == "" {
		return Config{}, errors.New("idmagic.realm_url must be an https:// url")
	}
	if config.IdMagicTLS, err = tlsConfig(file.IdMagic.CAFile, "", "idmagic.ca_file"); err != nil {
		return Config{}, err
	}
	if file.IdMagic.ProxyURL != "" {
		proxy, err := url.Parse(file.IdMagic.ProxyURL)
		if err != nil || (proxy.Scheme != "http" && proxy.Scheme != "https") || proxy.Host == "" {
			return Config{}, errors.New("idmagic.proxy_url must be an http:// or https:// url")
		}
		config.Proxy = proxy
	}
	if config.Dialect, err = domain.ParseDialect(file.LDAP.Dialect); err != nil {
		return Config{}, fmt.Errorf("ldap.dialect: %w", err)
	}
	if err := domain.ValidateBaseDNs(file.LDAP.BaseDNs); err != nil {
		return Config{}, err
	}
	config.Scope = domain.Scope{BaseDNs: file.LDAP.BaseDNs, Filter: file.LDAP.UserFilter}
	if config.Scope.Filter == "" {
		config.Scope.Filter = domain.DefaultUserFilter(config.Dialect)
	} else if err := ldap_goldap.ValidateFilter(config.Scope.Filter); err != nil {
		return Config{}, fmt.Errorf("ldap.user_filter: %w", err)
	}
	if config.Mapping, err = domain.EffectiveMapping(config.Dialect, file.LDAP.Attributes); err != nil {
		return Config{}, err
	}
	pageSize := defaultPageSize
	if file.LDAP.PageSize != nil {
		pageSize = *file.LDAP.PageSize
	}
	if pageSize < 1 || pageSize > maxPageSize {
		return Config{}, fmt.Errorf("ldap.page_size must be between 1 and %d", maxPageSize)
	}
	if config.Interval, err = duration(file.Sync.Interval, defaultInterval, "sync.interval"); err != nil {
		return Config{}, err
	}
	if config.FullInterval, err = duration(file.Sync.FullInterval, defaultFullInterval, "sync.full_interval"); err != nil {
		return Config{}, err
	}
	password, err := os.ReadFile(file.LDAP.BindPasswordFile)
	if err != nil {
		return Config{}, fmt.Errorf("ldap.bind_password_file: %w", err)
	}
	ldapTLS, err := tlsConfig(file.LDAP.CAFile, file.LDAP.ServerName, "ldap.ca_file")
	if err != nil {
		return Config{}, err
	}
	config.LDAP = ldap_goldap.Config{
		URL: file.LDAP.URL, StartTLS: file.LDAP.StartTLS, TLS: ldapTLS, Dialect: config.Dialect,
		BindDN: file.LDAP.BindDN, BindPassword: strings.TrimRight(string(password), "\r\n"),
		PageSize: uint32(pageSize), Timeout: time.Minute,
	}
	if _, err := ldap_goldap.New(config.LDAP); err != nil {
		return Config{}, err
	}
	return config, nil
}

func tlsConfig(caFile, serverName, field string) (*tls.Config, error) {
	config := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}
	if caFile == "" {
		return config, nil
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("%s: no PEM certificate", field)
	}
	config.RootCAs = pool
	return config, nil
}

func duration(value string, fallback time.Duration, field string) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration such as 5m", field)
	}
	return parsed, nil
}
