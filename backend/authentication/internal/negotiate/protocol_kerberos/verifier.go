// Package protocol_kerberos はチケットの暗号と SPNEGO の処理を既存の Kerberos 実装に委ねる。
package protocol_kerberos

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/ports"
	"github.com/otuschhoff/gokrb5/v8/gssapi"
	"github.com/otuschhoff/gokrb5/v8/keytab"
	"github.com/otuschhoff/gokrb5/v8/service"
	"github.com/otuschhoff/gokrb5/v8/spnego"
	"github.com/otuschhoff/gokrb5/v8/types"
)

type Verifier struct{}

func LoadCertificate(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("negotiate: TLS certificate is unavailable")
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("negotiate: TLS certificate must be PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

// CheckFiles は起動時に鍵と証明書の不備を検出する。要求時にも読み直すので、鍵の更新に再起動は要らない。
func CheckFiles(config domain.Config) error {
	kt, err := keytab.Load(config.KeytabFile)
	if err != nil {
		return fmt.Errorf("negotiate: keytab is unavailable or invalid")
	}
	found := false
	for _, p := range kt.Principals() {
		if p.Realm == config.Realm && (types.PrincipalName{NameString: p.Components}).PrincipalNameString() == config.ServicePrincipal {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("negotiate: configured SPN and realm are missing from keytab")
	}
	cert, err := LoadCertificate(config.CertificateFile)
	if err != nil {
		return err
	}
	return cert.VerifyHostname(config.ServicePrincipal[len("HTTP/"):])
}

func (Verifier) Verify(ctx context.Context, config domain.Config, token []byte, clientIP string) (ports.Verified, error) {
	if err := ctx.Err(); err != nil {
		return ports.Verified{}, err
	}
	if len(token) == 0 || len(token) > 64*1024 {
		return ports.Verified{}, fmt.Errorf("negotiate: invalid token length")
	}
	kt, err := keytab.Load(config.KeytabFile)
	if err != nil {
		return ports.Verified{}, fmt.Errorf("negotiate: keytab is unavailable")
	}
	certificate, err := LoadCertificate(config.CertificateFile)
	if err != nil {
		return ports.Verified{}, err
	}
	bindings, err := gssapi.TLSServerEndPoint(tls.ConnectionState{PeerCertificates: []*x509.Certificate{certificate}})
	if err != nil {
		return ports.Verified{}, err
	}
	var negotiation spnego.SPNEGOToken
	if err := negotiation.Unmarshal(token); err != nil || !negotiation.Init {
		return ports.Verified{}, fmt.Errorf("negotiate: invalid SPNEGO initial token")
	}
	var mechanism spnego.KRB5Token
	if err := mechanism.Unmarshal(negotiation.NegTokenInit.MechTokenBytes); err != nil || !mechanism.IsAPReq() {
		return ports.Verified{}, fmt.Errorf("negotiate: Kerberos AP_REQ is required")
	}
	request := &mechanism.APReq
	if request.Ticket.Realm != config.Realm || request.Ticket.SName.PrincipalNameString() != config.ServicePrincipal {
		return ports.Verified{}, fmt.Errorf("negotiate: service principal mismatch")
	}
	var address types.HostAddress
	if clientIP != "" {
		address, err = types.GetHostAddress(net.JoinHostPort(clientIP, "0"))
		if err != nil {
			return ports.Verified{}, err
		}
	}
	// 別に解析した AP_REQ は、共有の再送キーを暗号化前の authenticator から作るために使う。
	// この Verify は再送キャッシュを操作しない。SPNEGO と CBT の受理は直後の acceptor が行う。
	valid, err := request.Verify(kt, 5*time.Minute, address, nil)
	if err != nil || !valid {
		return ports.Verified{}, fmt.Errorf("negotiate: AP_REQ validation failed")
	}
	now := time.Now().UTC()
	if !request.Ticket.DecryptedEncPart.EndTime.After(now) || request.Authenticator.CRealm != config.Realm || len(request.Authenticator.CName.NameString) != 1 {
		return ports.Verified{}, fmt.Errorf("negotiate: ticket is expired or client realm is not accepted")
	}
	acceptor := spnego.SPNEGOService(kt,
		service.ServicePrincipals(config.ServicePrincipal+"@"+config.Realm),
		service.ClientAddress(address), service.MaxClockSkew(5*time.Minute),
		service.ExtendedProtection(service.ExtendedProtectionRequired), service.ChannelBindings(&bindings),
	)
	authenticated, _, status := acceptor.AcceptSecContext(&negotiation)
	if !authenticated || status.Code != gssapi.StatusComplete {
		return ports.Verified{}, fmt.Errorf("negotiate: SPNEGO or channel binding validation failed")
	}
	var reply []byte
	if response := acceptor.ResponseToken(); response != nil {
		reply, err = response.Marshal()
		if err != nil {
			return ports.Verified{}, err
		}
	}
	identity := domain.Principal{Realm: request.Authenticator.CRealm, Name: request.Authenticator.CName.NameString[0]}
	encoded, err := json.Marshal([]any{config.Realm, config.ServicePrincipal, identity.Name, request.Authenticator.CTime.Unix(), request.Authenticator.Cusec})
	if err != nil {
		return ports.Verified{}, err
	}
	digest := sha256.Sum256(encoded)
	return ports.Verified{Principal: identity, ReplayID: hex.EncodeToString(digest[:]), Reply: reply}, nil
}
