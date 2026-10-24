package config_file

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/protocol_kerberos"
)

type Configuration struct {
	AttemptSecret string          `json:"attempt_secret"`
	Automatic     bool            `json:"automatic"`
	Tenants       []domain.Config `json:"tenants"`
}

func Load(path string) (Configuration, []byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, nil, "", fmt.Errorf("negotiate: configuration is unavailable")
	}
	var config Configuration
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Configuration{}, nil, "", fmt.Errorf("negotiate: invalid configuration JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Configuration{}, nil, "", fmt.Errorf("negotiate: configuration must contain one JSON object")
	}
	secret, err := base64.StdEncoding.DecodeString(config.AttemptSecret)
	if err != nil || len(secret) < 32 {
		return Configuration{}, nil, "", fmt.Errorf("negotiate: attempt_secret must be at least 32 random bytes in base64")
	}
	config.AttemptSecret = ""
	for _, tenant := range config.Tenants {
		if err := tenant.Validate(); err != nil {
			return Configuration{}, nil, "", err
		}
		if err := protocol_kerberos.CheckFiles(tenant); err != nil {
			return Configuration{}, nil, "", err
		}
	}
	digest := sha256.Sum256(data)
	return config, secret, hex.EncodeToString(digest[:]), nil
}
