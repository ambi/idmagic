package support_http

import (
	"net"
	"net/netip"

	"github.com/labstack/echo/v5"
)

// ClientIPExtractor は、明示された範囲の代理だけを信頼して送信元を解決する。
func ClientIPExtractor(trusted []netip.Prefix) echo.IPExtractor {
	if len(trusted) == 0 {
		return echo.ExtractIPDirect()
	}
	options := []echo.TrustOption{
		echo.TrustLoopback(false), echo.TrustLinkLocal(false), echo.TrustPrivateNet(false),
	}
	for _, prefix := range trusted {
		options = append(options, echo.TrustIPRange(&net.IPNet{
			IP: net.IP(prefix.Addr().AsSlice()), Mask: net.CIDRMask(prefix.Bits(), prefix.Addr().BitLen()),
		}))
	}
	return echo.ExtractIPFromXFFHeader(options...)
}
