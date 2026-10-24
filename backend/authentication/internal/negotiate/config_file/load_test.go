package config_file_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/config_file"
	fixture "github.com/ambi/idmagic/backend/authentication/testing_kerberos"
)

//spec:covers REQ-AUTHENTICATION-042: 診断と起動は未知の設定、不正な秘密鍵、鍵とサービス名の不一致を認めず、設定の内容を返さない。
func TestTrustConfigurationValidatesFilesAndSecret(t *testing.T) {
	f := fixture.New(t)
	path := f.WriteConfiguration(t, "tenant", false)
	config, secret, fingerprint, err := config_file.Load(path)
	if err != nil || len(secret) != 32 || len(fingerprint) != 64 || config.AttemptSecret != "" || len(config.Tenants) != 1 {
		t.Fatalf("invalid configuration result: err=%v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"unknown", "secret", "spn", "keytab", "certificate", "trailing"} {
		t.Run(name, func(t *testing.T) {
			var input map[string]any
			if err := json.Unmarshal(original, &input); err != nil {
				t.Fatal(err)
			}
			tenant := input["tenants"].([]any)[0].(map[string]any)
			switch name {
			case "unknown":
				input["incorrect_option"] = true
			case "secret":
				input["attempt_secret"] = "c2hvcnQ="
			case "spn":
				tenant["service_principal"] = "HTTP/other.example.com"
			case "keytab":
				tenant["keytab_file"] = path + ".missing"
			case "certificate":
				tenant["certificate_file"] = path
			}
			data, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			if name == "trailing" {
				data = append(data, []byte(" {}")...)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, _, err := config_file.Load(path); err == nil {
				t.Fatal("invalid trust accepted")
			}
		})
	}
}
