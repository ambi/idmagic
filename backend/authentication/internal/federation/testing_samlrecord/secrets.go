package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// CheckSecrets は、記録が記録に使った資格情報と秘密鍵を含まないことを確かめる。保存の前に呼ぶ。
//
// 拒否するのは、管理者のパスワード、利用者のパスワード、管理者のアクセストークンのいずれかの値か、
// 秘密鍵の PEM を含む記録である。記録の本文は URL エンコードと base64 を経るので、値はそれぞれの表現でも探す。
func CheckSecrets(rec testing_httprecord.Recording, secrets ...string) error {
	encoded, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	var searched strings.Builder
	searched.Write(encoded)
	for _, ex := range rec.Exchanges {
		searched.WriteString("\n" + decodedForm(ex.Request.Body))
	}
	text := searched.String()
	var problems []string
	for _, secret := range secrets {
		if secret != "" && strings.Contains(text, secret) {
			problems = append(problems, "the recording contains a credential used to record it")
		}
	}
	if strings.Contains(text, "PRIVATE KEY") {
		problems = append(problems, "the recording contains a private key PEM")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// decodedForm は、フォームの本文の値と、base64 として読める値の中身を並べて返す。
func decodedForm(body json.RawMessage) string {
	values, err := url.ParseQuery(string(testing_httprecord.BodyBytes(body)))
	if err != nil {
		return ""
	}
	var out strings.Builder
	for _, list := range values {
		for _, value := range list {
			out.WriteString(value + "\n")
			if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
				out.Write(decoded)
				out.WriteString("\n")
			}
		}
	}
	return out.String()
}
