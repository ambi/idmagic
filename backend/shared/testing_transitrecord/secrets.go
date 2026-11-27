package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// RecordedToken は、記録の X-Vault-Token に置く値である。記録に使った root トークンはこの値へ置き換える。
const RecordedToken = "recorded-root-token"

// FixtureDEK は、記録の暗号化と復号に使う固定の DEK である。秘密ではないテスト用の値であり、記録に現れてよい平文はこれだけである。
var FixtureDEK = []byte("0123456789abcdef0123456789abcdef")

// CheckSecrets は、記録が鍵の材料と記録に使った資格情報を含まないことを確かめる。保存の前に呼ぶ。
//
// 拒否するのは、root トークンの値、秘密鍵の PEM、FixtureDEK 以外の `plaintext`、
// 鍵の材料を書き出す Transit の経路（export、backup）を含む記録である。
func CheckSecrets(rec testing_httprecord.Recording, token string) error {
	encoded, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	var problems []string
	if token != "" && strings.Contains(string(encoded), token) {
		problems = append(problems, "the recording contains the root token used to record it")
	}
	if strings.Contains(string(encoded), "PRIVATE KEY") {
		problems = append(problems, "the recording contains a private key PEM")
	}
	fixture := base64.StdEncoding.EncodeToString(FixtureDEK)
	for _, ex := range rec.Exchanges {
		if exportsKeyMaterial(ex.Request.Path) {
			problems = append(problems, fmt.Sprintf("%s: %s exports key material", ex.Name, ex.Request.Path))
		}
		for _, body := range []json.RawMessage{ex.Request.Body, ex.Response.Body} {
			for _, value := range plaintexts(body) {
				if value != fixture {
					problems = append(problems, fmt.Sprintf("%s: plaintext other than the fixture DEK", ex.Name))
				}
			}
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func exportsKeyMaterial(path string) bool {
	return strings.Contains(path, "/export/") || strings.Contains(path, "/backup/")
}

// plaintexts は、本文の中のすべての `plaintext` の文字列の値を返す。
func plaintexts(body json.RawMessage) []string {
	if len(body) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil
	}
	var found []string
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for key, child := range v {
				if s, ok := child.(string); ok && key == "plaintext" {
					found = append(found, s)
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(value)
	return found
}
