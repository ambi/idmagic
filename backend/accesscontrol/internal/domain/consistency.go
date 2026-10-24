package domain

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// EncodeConsistencyToken はテナントと書き込み版を束縛した不透明な整合トークンを返す。
// テナントを含めるので、他テナントで発行されたトークンを提示されても照合で弾ける。
func EncodeConsistencyToken(tenantID string, version int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(tenantID + ":" + strconv.FormatInt(version, 10)))
}

// DecodeConsistencyToken は tenantID に対して発行されたトークンの版を返す。
// 形式が壊れている、または別テナントのトークンである場合は fail-closed で拒否する。
//
// 受理するのは正規形、つまり EncodeConsistencyToken がそのテナントと版に対して返す文字列だけである。
// base64 の復号は入力中の改行を読み飛ばし、末尾の余剰ビットも既定では無視するので、復号できることを
// 条件にすると同じテナントと版を表す文字列が無数に通り、不透明であるはずのトークンが可鍛になる。
func DecodeConsistencyToken(token, tenantID string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("%w: malformed token", ErrConsistencyNotSatisfied)
	}
	issuedTenant, rawVersion, found := strings.Cut(string(raw), ":")
	if !found || issuedTenant != tenantID {
		return 0, fmt.Errorf("%w: token was not issued for this tenant", ErrConsistencyNotSatisfied)
	}
	version, err := strconv.ParseInt(rawVersion, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: malformed version", ErrConsistencyNotSatisfied)
	}
	if EncodeConsistencyToken(tenantID, version) != token {
		return 0, fmt.Errorf("%w: malformed token", ErrConsistencyNotSatisfied)
	}
	return version, nil
}
