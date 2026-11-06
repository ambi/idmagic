package tokens_jose

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"
)

// 再利用の期間の上限は、ヒントがないときの既定値と同じ 5 分である (SPIFFE Federation の 4.1 節)。
// kid が一致しないときに取得し直す仕組みを持たないため、長いヒントでもこれを超えない。
// 下限は、短すぎるヒントの取得元を呼び続けないためのもの。
const (
	spiffeBundleMaxReuse = 5 * time.Minute
	spiffeBundleMinReuse = 30 * time.Second
)

// spiffeSVIDUses は SPIFFE Trust Domain and Bundle の 4.2.2 節が定める use の値。どれか一つを
// 持つ鍵があれば、文書を SPIFFE バンドルとして読む。値は大文字と小文字を区別する。
var spiffeSVIDUses = []string{"x509-svid", "jwt-svid", "wit-svid"}

// SPIFFEBundleJWTKeys は、spiffe の信頼設定が取得した文書から JWT-SVID の署名の検証に使う鍵と、
// 鍵を再利用してよい期間を返す。
//
// SPIFFE バンドルなら use が jwt-svid の鍵だけを返し、use のない鍵も返さない (4.2.2 節は
// 無視を求める)。SPIFFE バンドルでない JWKS (SPIRE の OIDC Discovery Provider の形) なら、
// use が sig か省略の鍵を返す。使える鍵がなければ空を返し、エラーにはしない。
func SPIFFEBundleJWTKeys(document map[string]any) (keys []map[string]any, reuseFor time.Duration) {
	reuseFor = spiffeBundleReuseFor(document["spiffe_refresh_hint"])
	all, err := InlineJWKs(document)
	if err != nil {
		return []map[string]any{}, reuseFor
	}
	bundle := false
	for _, key := range all {
		if use, _ := key["use"].(string); slices.Contains(spiffeSVIDUses, use) {
			bundle = true
			break
		}
	}
	keys = []map[string]any{}
	for _, key := range all {
		use, present := key["use"]
		if bundle && use == "jwt-svid" || !bundle && (!present || use == "sig") {
			keys = append(keys, key)
		}
	}
	return keys, reuseFor
}

func spiffeBundleReuseFor(hint any) time.Duration {
	seconds, ok := hint.(float64)
	// 上限との比較を Duration への変換より先に置き、巨大な値の変換で桁あふれさせない。
	if !ok || seconds < 1 || seconds != math.Trunc(seconds) || seconds >= spiffeBundleMaxReuse.Seconds() {
		return spiffeBundleMaxReuse
	}
	return max(time.Duration(seconds)*time.Second, spiffeBundleMinReuse)
}

// ResolveSPIFFEBundle は spiffe の信頼設定の鍵を、インラインの文書があればそこから、なければ
// jwksURI から SPIFFEBundleJWTKeys の規則で選んで返す。jwksURI の取得は ResolveJWKSSource と
// 同じ SSRF の防御と Web PKI の検証 (Bundle Endpoint の https_web) を通り、結果は
// SPIFFEBundleJWTKeys が返す期間だけ再利用する。
func (r *JWKResolver) ResolveSPIFFEBundle(ctx context.Context, jwksURI *string, inline map[string]any) ([]map[string]any, error) {
	if inline != nil {
		if _, err := InlineJWKs(inline); err == nil {
			keys, _ := SPIFFEBundleJWTKeys(inline)
			return keys, nil
		}
	}
	if jwksURI == nil || *jwksURI == "" {
		return nil, errors.New("no jwks or jwks_uri configured")
	}
	return r.fetchDocument(ctx, *jwksURI, spiffeBundleCacheKeyPrefix, func(document map[string]any) ([]map[string]any, time.Duration, error) {
		// keys の配列を持たない応答は JWKS ではないので、通常の JWKS と同じく取得の失敗とし、
		// 直近に取得できた鍵 (WorkloadJWKSCache) を空の集合で上書きしない。
		if _, err := InlineJWKs(document); err != nil {
			return nil, 0, err
		}
		keys, reuseFor := SPIFFEBundleJWTKeys(document)
		return keys, reuseFor, nil
	})
}

// 同じ URI を oidc と spiffe の信頼設定が使っても、読み方の違う結果を共有しない。
const spiffeBundleCacheKeyPrefix = "spiffe-bundle:"
