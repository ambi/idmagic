package push_http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func lookupTo(addrs ...string) func(context.Context, string) ([]netip.Addr, error) {
	return func(context.Context, string) ([]netip.Addr, error) {
		out := make([]netip.Addr, len(addrs))
		for i, addr := range addrs {
			out[i] = netip.MustParseAddr(addr)
		}
		return out, nil
	}
}

//spec:covers REQ-SHAREDSIGNALS-017: 解決先に一つでも非公開があれば接続しない。
//spec:covers EX-SHAREDSIGNALS-017-02: 登録後に非公開へ解決が変わった宛先へ接続しない。
func TestPusherRefusesNonPublicResolution(t *testing.T) {
	connected := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { connected = true }))
	defer server.Close()
	port := server.URL[strings.LastIndex(server.URL, ":"):]
	for name, addrs := range map[string][]string{"loopback": {"127.0.0.1"}, "mixed": {"8.8.8.8", "127.0.0.1"}, "metadata": {"169.254.169.254"}, "cgnat": {"100.100.1.1"}} {
		t.Run(name, func(t *testing.T) {
			p := newPusher(lookupTo(addrs...))
			err := p.Push(t.Context(), "https://hooks.example.com"+port+"/in", "", "set")
			if !errors.Is(err, errNonPublicDestination) {
				t.Fatalf("Push = %v", err)
			}
			if err := p.CheckDestination(t.Context(), "https://hooks.example.com/"); !errors.Is(err, ssdomain.ErrDeliveryEndpointNotAllowed) {
				t.Fatalf("CheckDestination = %v", err)
			}
		})
	}
	if connected {
		t.Fatal("connected to a non-public destination")
	}
}

//spec:covers REQ-SHAREDSIGNALS-017: 環境のプロキシを使わず、接続を再利用せず、リダイレクトに従わない。
func TestProductionClientRefusesBypasses(t *testing.T) {
	p := newPusher(lookupTo("8.8.8.8"))
	transport := p.Client.Transport.(*http.Transport)
	if transport.Proxy != nil || !transport.DisableKeepAlives {
		t.Fatalf("proxy set=%v keepalives disabled=%v", transport.Proxy != nil, transport.DisableKeepAlives)
	}
	followed := false
	final := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { followed = true }))
	defer final.Close()
	first := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, final.URL, http.StatusFound) }))
	defer first.Close()
	client := first.Client()
	client.CheckRedirect = p.Client.CheckRedirect
	testPusher := &HTTPSecurityEventPusher{Client: client}
	if err := testPusher.Push(t.Context(), first.URL, "", "set"); err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("Push = %v", err)
	}
	if followed {
		t.Fatal("followed a redirect")
	}
}

func TestTransportErrorDoesNotLeakAddress(t *testing.T) {
	p := newPusher(lookupTo("8.8.8.8"))
	p.Client.Transport.(*http.Transport).DialContext = func(context.Context, string, string) (netConn, error) {
		return nil, errors.New("dial tcp 10.0.0.1:443: refused")
	}
	if err := p.Push(t.Context(), "https://hooks.example.com/", "", "set"); err == nil || strings.Contains(err.Error(), "10.0.0.1") {
		t.Fatalf("Push = %v", err)
	}
}

type netConn = net.Conn
