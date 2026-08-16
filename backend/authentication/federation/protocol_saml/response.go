// Package protocol_saml implements the Authentication broker's upstream SAML SP port.
package protocol_saml

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const maxSAMLResponseBytes = 1 << 20

func BuildAuthnRequest(
	connection domain.IdentityProviderConnection,
	attempt domain.FederatedLoginAttempt,
	acsURL string,
	now time.Time,
) (string, error) {
	if !connection.Active() || connection.Protocol != domain.ProtocolSAML {
		return "", errors.New("SAML connection is not active")
	}
	if attempt.State == "" || attempt.RequestID == "" {
		return "", errors.New("RelayState and request ID are required")
	}
	now = normalizedNow(now)
	request := etree.NewElement("samlp:AuthnRequest")
	request.CreateAttr("xmlns:samlp", "urn:oasis:names:tc:SAML:2.0:protocol")
	request.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
	request.CreateAttr("ID", attempt.RequestID)
	request.CreateAttr("Version", "2.0")
	request.CreateAttr("IssueInstant", now.Format(time.RFC3339))
	request.CreateAttr("Destination", connection.SAMLSSOURL)
	request.CreateAttr("AssertionConsumerServiceURL", acsURL)
	request.CreateAttr("ProtocolBinding", "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST")
	request.CreateElement("saml:Issuer").SetText(acsURL)
	document := etree.NewDocument()
	document.SetRoot(request)
	raw, err := document.WriteToBytes()
	if err != nil {
		return "", err
	}
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	if err != nil {
		return "", err
	}
	if _, err := writer.Write(raw); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	values := url.Values{
		"SAMLRequest": {base64.StdEncoding.EncodeToString(compressed.Bytes())},
		"RelayState":  {attempt.State},
	}
	return connection.SAMLSSOURL + "?" + values.Encode(), nil
}

func ValidateResponse(
	ctx context.Context,
	connection domain.IdentityProviderConnection,
	attempt domain.FederatedLoginAttempt,
	encodedResponse, acsURL string,
	now time.Time,
	replay federationports.ReplayStore,
) (domain.NormalizedClaims, error) {
	var empty domain.NormalizedClaims
	if !connection.Active() || connection.Protocol != domain.ProtocolSAML {
		return empty, errors.New("SAML connection is not active")
	}
	raw, err := base64.StdEncoding.DecodeString(encodedResponse)
	if err != nil || len(raw) == 0 || len(raw) > maxSAMLResponseBytes {
		return empty, errors.New("invalid SAML response encoding or size")
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(raw); err != nil {
		return empty, fmt.Errorf("parse SAML response: %w", err)
	}
	root := document.Root()
	if root == nil || root.Tag != "Response" {
		return empty, errors.New("SAML Response root is required")
	}
	responseID := strings.TrimSpace(root.SelectAttrValue("ID", ""))
	if responseID == "" || hasDuplicateIDs(root) {
		return empty, errors.New("SAML response IDs are missing or duplicated")
	}
	// ID は replay 表の主キーの成分になる。上限が無いと、外部 IdP が決めた長さの
	// まま btree の索引行上限で落ちる。応答形は SAML が決めるので、型付きの
	// LengthError は返さない。
	if err := spec.CheckKeyString("ID", responseID, spec.LengthProtocolMessageID, spec.BytesProtocolMessageID); err != nil {
		return empty, fmt.Errorf("SAML Response %s", err.Error())
	}
	validated, err := validateSignature(root, connection.SAMLSigningCertificates)
	if err != nil {
		return empty, err
	}
	if validated.Tag != "Response" || validated.SelectAttrValue("ID", "") != responseID {
		return empty, errors.New("SAML signature did not validate the response root")
	}
	if validated.SelectAttrValue("Destination", "") != acsURL ||
		validated.SelectAttrValue("InResponseTo", "") != attempt.RequestID {
		return empty, errors.New("SAML response correlation mismatch")
	}
	if textOf(validated, "Issuer") != connection.SAMLEntityID {
		return empty, errors.New("SAML issuer mismatch")
	}
	status := childByLocal(childByLocal(validated, "Status"), "StatusCode")
	if status == nil || status.SelectAttrValue("Value", "") != "urn:oasis:names:tc:SAML:2.0:status:Success" {
		return empty, errors.New("SAML response status is not success")
	}
	assertion := childByLocal(validated, "Assertion")
	if assertion == nil {
		return empty, errors.New("SAML assertion is required")
	}
	if textOf(assertion, "Issuer") != connection.SAMLEntityID {
		return empty, errors.New("SAML assertion issuer mismatch")
	}
	now = normalizedNow(now)
	if err := validateConditions(assertion, acsURL, now); err != nil {
		return empty, err
	}
	if err := validateSubjectConfirmation(assertion, attempt.RequestID, acsURL, now); err != nil {
		return empty, err
	}
	if replay == nil {
		return empty, errors.New("SAML replay store is required")
	}
	reserved, err := replay.Reserve(ctx, connection.TenantID, responseID, now.Add(10*time.Minute))
	if err != nil {
		return empty, err
	}
	if !reserved {
		return empty, errors.New("SAML response replay detected")
	}
	values := attributeValues(assertion)
	values["NameID"] = textOf(childByLocal(assertion, "Subject"), "NameID")
	return normalizeClaims(connection.ClaimMapping, values)
}

// ValidateSigningCertificates reports, without a browser round-trip, whether the connection's
// saved SAML signing certificates parse as valid X.509 and are within their validity period.
// An empty slice means every configured certificate is currently usable.
func ValidateSigningCertificates(certificates []string, now time.Time) []string {
	if len(certificates) == 0 {
		return []string{"no SAML signing certificates are configured"}
	}
	now = normalizedNow(now)
	var failures []string
	for _, certificatePEM := range certificates {
		block, _ := pem.Decode([]byte(certificatePEM))
		if block == nil || block.Type != "CERTIFICATE" {
			failures = append(failures, "a SAML signing certificate could not be parsed")
			continue
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			failures = append(failures, "a SAML signing certificate could not be parsed")
			continue
		}
		if now.Before(certificate.NotBefore) || !now.Before(certificate.NotAfter) {
			failures = append(failures, "a SAML signing certificate is outside its validity period")
		}
	}
	return failures
}

func validateSignature(root *etree.Element, certificates []string) (*etree.Element, error) {
	if childByLocal(root, "Signature") == nil {
		return nil, errors.New("SAML response signature is required")
	}
	var lastErr error
	for _, certificatePEM := range certificates {
		block, _ := pem.Decode([]byte(certificatePEM))
		if block == nil || block.Type != "CERTIFICATE" {
			lastErr = errors.New("invalid SAML signing certificate")
			continue
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			lastErr = err
			continue
		}
		store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{certificate}}
		validation := dsig.NewDefaultValidationContext(store)
		validation.IdAttribute = "ID"
		validated, err := validation.Validate(root)
		if err == nil {
			return validated, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("invalid SAML response signature: %w", lastErr)
}

func validateConditions(assertion *etree.Element, audience string, now time.Time) error {
	conditions := childByLocal(assertion, "Conditions")
	if conditions == nil {
		return errors.New("SAML conditions are required")
	}
	if raw := conditions.SelectAttrValue("NotBefore", ""); raw != "" {
		notBefore, err := time.Parse(time.RFC3339, raw)
		if err != nil || now.Add(time.Minute).Before(notBefore) {
			return errors.New("SAML assertion is not yet valid")
		}
	}
	notOnOrAfter, err := time.Parse(time.RFC3339, conditions.SelectAttrValue("NotOnOrAfter", ""))
	if err != nil || !now.Before(notOnOrAfter) {
		return errors.New("SAML assertion expired")
	}
	restriction := childByLocal(conditions, "AudienceRestriction")
	if restriction == nil || textOf(restriction, "Audience") != audience {
		return errors.New("SAML audience mismatch")
	}
	return nil
}

func validateSubjectConfirmation(assertion *etree.Element, requestID, acsURL string, now time.Time) error {
	subject := childByLocal(assertion, "Subject")
	confirmation := childByLocal(subject, "SubjectConfirmation")
	data := childByLocal(confirmation, "SubjectConfirmationData")
	if confirmation == nil || data == nil ||
		confirmation.SelectAttrValue("Method", "") != "urn:oasis:names:tc:SAML:2.0:cm:bearer" ||
		data.SelectAttrValue("Recipient", "") != acsURL ||
		data.SelectAttrValue("InResponseTo", "") != requestID {
		return errors.New("SAML subject confirmation mismatch")
	}
	expires, err := time.Parse(time.RFC3339, data.SelectAttrValue("NotOnOrAfter", ""))
	if err != nil || !now.Before(expires) {
		return errors.New("SAML subject confirmation expired")
	}
	return nil
}

func normalizeClaims(mapping domain.ClaimMapping, values map[string]string) (domain.NormalizedClaims, error) {
	out := domain.NormalizedClaims{
		Subject:       strings.TrimSpace(values[mapping.Subject]),
		Username:      strings.TrimSpace(values[mapping.Username]),
		Email:         strings.TrimSpace(values[mapping.Email]),
		Name:          strings.TrimSpace(values[mapping.Name]),
		EmailVerified: strings.EqualFold(strings.TrimSpace(values[mapping.EmailVerified]), "true"),
	}
	if out.Subject == "" || out.Username == "" {
		return domain.NormalizedClaims{}, errors.New("required mapped SAML attributes are missing")
	}
	return out, nil
}

func attributeValues(assertion *etree.Element) map[string]string {
	values := map[string]string{}
	statement := childByLocal(assertion, "AttributeStatement")
	if statement == nil {
		return values
	}
	for _, attribute := range statement.ChildElements() {
		if attribute.Tag != "Attribute" {
			continue
		}
		values[attribute.SelectAttrValue("Name", "")] = textOf(attribute, "AttributeValue")
	}
	return values
}

func hasDuplicateIDs(root *etree.Element) bool {
	seen := map[string]struct{}{}
	var walk func(*etree.Element) bool
	walk = func(element *etree.Element) bool {
		if id := element.SelectAttrValue("ID", ""); id != "" {
			if _, exists := seen[id]; exists {
				return true
			}
			seen[id] = struct{}{}
		}
		return slices.ContainsFunc(element.ChildElements(), walk)
	}
	return walk(root)
}

func childByLocal(parent *etree.Element, local string) *etree.Element {
	if parent == nil {
		return nil
	}
	for _, child := range parent.ChildElements() {
		if child.Tag == local {
			return child
		}
	}
	return nil
}

func textOf(parent *etree.Element, local string) string {
	if child := childByLocal(parent, local); child != nil {
		return strings.TrimSpace(child.Text())
	}
	return ""
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
