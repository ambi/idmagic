// Package realm_http は、Microsoft のサインインの画面が使う公開のレルム探索に、ドメインの送り先を問い合わせる。
//
// エンドポイント（getuserrealm.srf）は Microsoft の公式の API の文書にない。応答は判定に使う
// NameSpaceType と AuthURL だけを読み、失敗はエラーで返して、判定の側が確かめられないものとして扱う。
package realm_http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/safehttp"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

const (
	// DefaultURL は Microsoft の商用クラウドのレルム探索である。
	DefaultURL = "https://login.microsoftonline.com/getuserrealm.srf"
	// realmCheckLocalPart は、問い合わせの login のローカル部である。探索はドメインで送り先を決めるので、
	// 存在しない利用者でよく、利用者の名前を Microsoft へ送らない。
	realmCheckLocalPart = "idmagic-realm-check"
	maxResponseBytes    = 64 << 10
	dialTimeout         = 2 * time.Second
	tlsTimeout          = 2 * time.Second
	requestTimeout      = 3 * time.Second
)

// Discovery は EntraRealmDiscovery の HTTP の実装である。URL が空なら探索を無効にする。
type Discovery struct {
	endpoint *url.URL
	client   *http.Client
}

var _ wsfedports.EntraRealmDiscovery = Discovery{}

// New は、運用者が設定した探索の URL から Discovery を作る。https に限り、試験のために loopback の
// http だけを許す。利用者情報、問い合わせ文字列、断片を持つ URL は拒否する。
func New(rawURL string) (Discovery, error) {
	if rawURL == "" {
		return Discovery{}, nil
	}
	endpoint, err := url.Parse(rawURL)
	if err != nil {
		return Discovery{}, fmt.Errorf("entra realm discovery URL: %w", err)
	}
	if endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return Discovery{}, errors.New("entra realm discovery URL must be an absolute URL without userinfo, query, or fragment")
	}
	switch {
	case endpoint.Scheme == "https":
		return Discovery{endpoint: endpoint, client: safehttp.NewClient(safehttp.Config{
			DialTimeout: dialTimeout, TLSTimeout: tlsTimeout, RequestTimeout: requestTimeout,
			// MaxRedirects が 0 なので、どのリダイレクトにも従わない。ValidateURL は最初の要求にも働く。
			MaxRedirects: 0, ValidateURL: requireHTTPS,
		})}, nil
	case endpoint.Scheme == "http" && isLoopback(endpoint.Hostname()):
		return Discovery{endpoint: endpoint, client: &http.Client{
			Timeout:       requestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirects are not followed") },
		}}, nil
	}
	return Discovery{}, errors.New("entra realm discovery URL must use https (http only for loopback)")
}

func requireHTTPS(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		return errors.New("entra realm discovery requires https")
	}
	return nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Discover は、ドメインの送り先を問い合わせる。無効なら wsfedports.ErrEntraRealmDiscoveryDisabled を返す。
func (d Discovery) Discover(ctx context.Context, domain string) (feddomain.EntraDomainRealm, error) {
	if d.endpoint == nil {
		return feddomain.EntraDomainRealm{}, wsfedports.ErrEntraRealmDiscoveryDisabled
	}
	target := *d.endpoint
	target.RawQuery = url.Values{"login": {realmCheckLocalPart + "@" + domain}, "json": {"1"}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), http.NoBody)
	if err != nil {
		return feddomain.EntraDomainRealm{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := d.client.Do(request)
	if err != nil {
		return feddomain.EntraDomainRealm{}, fmt.Errorf("entra realm discovery: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return feddomain.EntraDomainRealm{}, fmt.Errorf("entra realm discovery: status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return feddomain.EntraDomainRealm{}, fmt.Errorf("entra realm discovery: %w", err)
	}
	if len(body) > maxResponseBytes {
		return feddomain.EntraDomainRealm{}, errors.New("entra realm discovery: the response is too large")
	}
	var realm struct {
		NameSpaceType string `json:"NameSpaceType"`
		AuthURL       string `json:"AuthURL"`
	}
	if err := json.Unmarshal(body, &realm); err != nil {
		return feddomain.EntraDomainRealm{}, fmt.Errorf("entra realm discovery: %w", err)
	}
	return feddomain.EntraDomainRealm{NamespaceType: realm.NameSpaceType, AuthURL: realm.AuthURL}, nil
}
