package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

func exchange(name, path, reqBody, respBody string) testing_httprecord.Exchange {
	ex := testing_httprecord.Exchange{Name: name}
	ex.Request.Method = "POST"
	ex.Request.Path = path
	ex.Request.Headers = map[string]string{"X-Vault-Token": RecordedToken}
	if reqBody != "" {
		ex.Request.Body = json.RawMessage(reqBody)
	}
	ex.Response.Status = 200
	if respBody != "" {
		ex.Response.Body = json.RawMessage(respBody)
	}
	return ex
}

// 保存前の検査は、鍵の材料と使った資格情報を含む記録を拒否し、固定の DEK だけを含む記録を受け入れる。
func TestCheckSecrets(t *testing.T) {
	const token = "s.4f1c9a0e2b7d4e6f"
	fixture := base64.StdEncoding.EncodeToString(FixtureDEK)
	other := base64.StdEncoding.EncodeToString([]byte("a real data key from somewhere"))
	tests := []struct {
		name string
		ex   testing_httprecord.Exchange
		want string // 空なら受け入れる
	}{
		{"固定の DEK の暗号化と復号", exchange("encrypt", "/v1/transit/encrypt/k", `{"plaintext":"`+fixture+`"}`, `{"data":{"plaintext":"`+fixture+`"}}`), ""},
		{"公開鍵", exchange("read", "/v1/transit/keys/k", "", `{"data":{"keys":{"1":{"public_key":"-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----\n"}}}}`), ""},
		{"使った root トークンが応答に残る", exchange("token", "/v1/auth/token/lookup-self", "", `{"data":{"id":"`+token+`"}}`), "root token"},
		{"使った root トークンがヘッダーに残る", func() testing_httprecord.Exchange {
			ex := exchange("header", "/v1/transit/keys/k", "", "")
			ex.Request.Headers["X-Vault-Token"] = token
			return ex
		}(), "root token"},
		{"秘密鍵の PEM", exchange("export", "/v1/transit/keys/k", "", `{"data":{"keys":{"1":"-----BEGIN RSA PRIVATE KEY-----\nMIIE\n-----END RSA PRIVATE KEY-----\n"}}}`), "private key"},
		{"固定でない平文の DEK を応答が返す", exchange("decrypt", "/v1/transit/decrypt/k", `{"ciphertext":"vault:v1:AAAA"}`, `{"data":{"plaintext":"`+other+`"}}`), "plaintext"},
		{"固定でない平文の DEK を要求が送る", exchange("encrypt", "/v1/transit/encrypt/k", `{"plaintext":"`+other+`"}`, ""), "plaintext"},
		{"データ鍵の生成が平文の鍵を返す", exchange("datakey", "/v1/transit/datakey/plaintext/k", "", `{"data":{"ciphertext":"vault:v1:AAAA","plaintext":"`+other+`"}}`), "plaintext"},
		{"鍵を書き出す経路", exchange("export", "/v1/transit/export/encryption-key/k", "", `{"data":{}}`), "exports key material"},
		{"鍵のバックアップ", exchange("backup", "/v1/transit/backup/k", "", `{"data":{"backup":"eyJ"}}`), "exports key material"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := testing_httprecord.Recording{Exchanges: []testing_httprecord.Exchange{tc.ex}}
			err := CheckSecrets(rec, token)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("CheckSecrets() = %v, want the recording accepted", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CheckSecrets() = %v, want an error about %q", err, tc.want)
			}
		})
	}
}
