package handlers_http

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/internal/usecases"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"

	"github.com/labstack/echo/v5"
)

// maxEntraObservationBytes は、観測文書の本文の上限である。管理 API の JSON の本文と同じ上限にする。
const maxEntraObservationBytes = 64 << 10

// entraVerification は、Deps の依存から Entra の反映の確認のユースケースを組み立てる。
func (d Deps) entraVerification() wsfedusecases.EntraFederationVerification {
	var realms wsfedports.EntraRealmDiscovery = disabledRealmDiscovery{}
	if d.EntraRealms != nil {
		realms = d.EntraRealms
	}
	return wsfedusecases.EntraFederationVerification{
		RPRepo: d.WsFedRPRepo, Observations: d.EntraObservations, Certificates: signerCertificates{d}, Realms: realms,
	}
}

// disabledRealmDiscovery は、探索を配線していない構成で、解除を観測記録だけで判定させる。
type disabledRealmDiscovery struct{}

func (disabledRealmDiscovery) Discover(context.Context, string) (feddomain.EntraDomainRealm, error) {
	return feddomain.EntraDomainRealm{}, wsfedports.ErrEntraRealmDiscoveryDisabled
}

// signerCertificates は、XML 署名の提供元を Entra に登録すべき証明書として読む。
type signerCertificates struct{ d Deps }

func (s signerCertificates) Current(ctx context.Context) (*x509.Certificate, error) {
	if s.d.FederationSigner == nil {
		return nil, fmt.Errorf("federation signing certificate unavailable")
	}
	signer, err := s.d.FederationSigner.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	if signer == nil || signer.Certificate() == nil {
		return nil, fmt.Errorf("federation signing certificate unavailable")
	}
	return signer.Certificate(), nil
}

func (s signerCertificates) Next(ctx context.Context) (*x509.Certificate, error) {
	return s.d.FederationSigner.PreparedCertificate(ctx)
}

type entraDomainSummary struct {
	feddomain.EntraDomainFederation
	Verification feddomain.EntraVerification `json:"verification"`
}

func (d Deps) handleListEntraFederationDomains(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	views, err := d.entraVerification().Domains(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	domains := make([]entraDomainSummary, 0, len(views))
	for _, view := range views {
		domains = append(domains, entraDomainSummary{EntraDomainFederation: view.Domain, Verification: view.Verification})
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"domains": domains})
}

type entraCertificateSummary struct {
	SHA256Thumbprint string    `json:"sha256_thumbprint"`
	NotAfter         time.Time `json:"not_after"`
}

func certificateSummary(certificate *x509.Certificate) *entraCertificateSummary {
	if certificate == nil {
		return nil
	}
	return &entraCertificateSummary{
		SHA256Thumbprint: feddomain.CertificateThumbprint(certificate), NotAfter: certificate.NotAfter.UTC(),
	}
}

func (d Deps) handleGetEntraFederationDomain(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	view, err := d.entraVerification().Domain(c.Request().Context(), support.RequestTenantID(c), c.Param("domain_name"))
	if errors.Is(err, wsfedusecases.ErrEntraDomainNotConfigured) {
		return writeEntraDomainNotFound(c)
	}
	if err != nil {
		return err
	}
	unfederated, err := d.unfederatedSourceUsers(c, *view.RelyingParty.EntraProfile)
	if err != nil {
		return err
	}
	settings := view.Expected.GraphParameters()
	replyURL := ""
	if len(view.RelyingParty.ReplyURLs) > 0 {
		replyURL = view.RelyingParty.ReplyURLs[0]
	}
	certificates := map[string]any{"signing": certificateSummary(view.Expected.SigningCertificate)}
	if next := certificateSummary(view.Expected.NextSigningCertificate); next != nil {
		certificates["next"] = next
	}
	body := map[string]any{
		"domain":            view.Domain,
		"profile":           view.RelyingParty.EntraProfile,
		"reply_url":         replyURL,
		"powershell":        settings,
		"commands":          entraPowerShellCommands(view.Domain.Domain, settings),
		"procedure":         entraPowerShellProcedure(view.Domain.Domain, view.Domain.EntraTenantID, settings),
		"certificates":      certificates,
		"verification":      view.Verification,
		"known_limitations": []map[string]string{{"code": "hybrid_join_not_supported"}},
	}
	if unfederated.UserCount > 0 {
		body["unfederated_source_users"] = unfederated
	}
	return support.NoStoreJSON(c, http.StatusOK, body)
}

func (d Deps) handleRecordEntraFederationObservation(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request().Body, maxEntraObservationBytes+1))
	if err != nil {
		return err
	}
	if len(raw) > maxEntraObservationBytes {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The observation document exceeds 64 KiB.")
	}
	observation, err := feddomain.DecodeEntraFederationObservation(raw)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	verification, err := d.entraVerification().Record(c.Request().Context(), support.RequestTenantID(c), c.Param("domain_name"), observation, time.Now().UTC())
	switch {
	case errors.Is(err, wsfedusecases.ErrEntraDomainNotConfigured):
		return writeEntraDomainNotFound(c)
	case errors.Is(err, feddomain.ErrInvalidEntraObservation):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, wsfedports.ErrEntraObservationOutdated):
		return support.WriteProblem(c, http.StatusConflict, "observation_outdated", "The observation was retrieved before the stored one.")
	case err != nil:
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"verification": verification})
}

func (d Deps) handleRemoveEntraFederationDomain(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	result, err := d.entraVerification().Remove(c.Request().Context(), support.RequestTenantID(c), c.Param("domain_name"), time.Now().UTC())
	switch {
	case errors.Is(err, wsfedusecases.ErrEntraDomainNotConfigured):
		return writeEntraDomainNotFound(c)
	case errors.Is(err, feddomain.ErrEntraFederationStillActive):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "entra_federation_still_active",
			"Microsoft still sends this domain to IdMagic. Roll the domain back in Entra and retry; the conversion to Managed can take up to 60 minutes.")
	case errors.Is(err, feddomain.ErrEntraRollbackUnconfirmed):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "entra_rollback_unconfirmed",
			"The rollback in Entra could not be confirmed with Microsoft. Import a read-back observation to remove the domain on its basis.")
	case err != nil:
		return err
	}
	d.emit(result.Event)
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"domain": result.Domain.Domain, "remaining_domains": result.RemainingDomains,
		"relying_party_deleted": result.RelyingPartyDeleted, "confirmed_by": result.ConfirmedBy,
	})
}

func writeEntraDomainNotFound(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "not_found", "The domain is not configured for Entra federation.")
}
