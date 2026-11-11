package handlers_http

import (
	"slices"
	"strings"
)

type powerShellCommands struct {
	Create string `json:"create"`
	Update string `json:"update"`
}

func entraPowerShellCommands(domain string, settings map[string]string) powerShellCommands {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parameters strings.Builder
	for _, key := range keys {
		parameters.WriteString(" -" + key + " " + powerShellLiteral(settings[key]))
	}
	domainParameter := " -DomainId " + powerShellLiteral(domain)
	return powerShellCommands{
		Create: "New-MgDomainFederationConfiguration" + domainParameter + parameters.String(),
		Update: "Update-MgDomainFederationConfiguration" + domainParameter +
			" -InternalDomainFederationId (Get-MgDomainFederationConfiguration" + domainParameter + " -ErrorAction Stop).Id" + parameters.String(),
	}
}

// PowerShell は曲がった引用符も区切りとして読む。単一引用符のリテラルでは、
// 同じ引用符を二つ続けると、変数展開せず元の一文字として渡せる。
func powerShellLiteral(value string) string {
	return "'" + strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’", "‚", "‚‚", "‛", "‛‛").Replace(value) + "'"
}

// entraProcedure は、管理者が Graph PowerShell で順に実行する手順である。
type entraProcedure struct {
	Connect         string `json:"connect"`
	ConnectReadOnly string `json:"connect_read_only"`
	Backup          string `json:"backup"`
	Compare         string `json:"compare"`
	Apply           string `json:"apply"`
	ReadBack        string `json:"read_back"`
}

// Graph の最小権限は Microsoft Learn の各 API の Permissions の表に合わせる。構成の作成と更新は
// Domain-InternalFederation.ReadWrite.All、構成の一覧は .Read.All、ドメインの取得は Domain.Read.All を求める。
const (
	graphModule            = "Microsoft.Graph.Identity.DirectoryManagement"
	graphWriteScopes       = "'Domain.Read.All','Domain-InternalFederation.ReadWrite.All'"
	graphReadOnlyScopes    = "'Domain.Read.All','Domain-InternalFederation.Read.All'"
	entraObservationFormat = "idmagic.entra-federation-observation.v1"
)

func entraPowerShellProcedure(domain, entraTenantID string, settings map[string]string) entraProcedure {
	domainID := powerShellLiteral(domain)
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parameters, expected strings.Builder
	for _, key := range keys {
		parameters.WriteString(" -" + key + " " + powerShellLiteral(settings[key]))
		expected.WriteString("  " + powerShellLiteral(key) + " = " + powerShellLiteral(settings[key]) + "\n")
	}
	existing := "$existing = @(Get-MgDomainFederationConfiguration -DomainId " + domainID + " -ErrorAction Stop)"
	return entraProcedure{
		Connect:         entraConnect(domainID, entraTenantID, graphWriteScopes),
		ConnectReadOnly: entraConnect(domainID, entraTenantID, graphReadOnlyScopes),
		Backup: lines(
			"$stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ')",
			"$path = Join-Path (Get-Location) ('idmagic-entra-' + "+domainID+" + '-before-' + $stamp + '.json')",
			"# Only the federation configuration of the domain is saved. It holds no user data or credentials.",
			existing,
			"[System.IO.File]::WriteAllText($path, (ConvertTo-Json -InputObject $existing -Depth 5))",
			"$path",
		),
		Compare: lines(
			"$expected = [ordered]@{",
			strings.TrimSuffix(expected.String(), "\n"),
			"}",
			existing,
			"if ($existing.Count -eq 0) { 'No federation configuration exists. The apply step creates it.' } else {",
			"  foreach ($key in $expected.Keys) {",
			"    $current = [string]$existing[0].$key",
			"    if ($current -cne $expected[$key]) { [pscustomobject]@{ Setting = $key; Current = $current; Expected = $expected[$key] } }",
			"  }",
			"}",
		),
		Apply: lines(
			existing,
			"if ($existing.Count -gt 0) {",
			"  Update-MgDomainFederationConfiguration -DomainId "+domainID+" -InternalDomainFederationId $existing[0].Id"+parameters.String(),
			"} else {",
			"  New-MgDomainFederationConfiguration -DomainId "+domainID+parameters.String(),
			"}",
		),
		ReadBack: lines(
			"$domain = Get-MgDomain -DomainId "+domainID+" -ErrorAction Stop",
			"$configurations = @(Get-MgDomainFederationConfiguration -DomainId "+domainID+" -ErrorAction Stop)",
			"$observation = [ordered]@{",
			"  format = '"+entraObservationFormat+"'",
			"  entra_tenant_id = [string](Get-MgContext).TenantId",
			"  domain = [string]$domain.Id",
			"  authentication_type = [string]$domain.AuthenticationType",
			"  retrieved_at = (Get-Date).ToUniversalTime().ToString('o')",
			"  federation_configurations = @($configurations | ForEach-Object { [ordered]@{",
			"    issuer_uri = [string]$_.IssuerUri; passive_sign_in_uri = [string]$_.PassiveSignInUri",
			"    active_sign_in_uri = [string]$_.ActiveSignInUri; metadata_exchange_uri = [string]$_.MetadataExchangeUri",
			"    sign_out_uri = [string]$_.SignOutUri; signing_certificate = [string]$_.SigningCertificate",
			"    next_signing_certificate = [string]$_.NextSigningCertificate",
			"    preferred_authentication_protocol = [string]$_.PreferredAuthenticationProtocol",
			"    federated_idp_mfa_behavior = [string]$_.FederatedIdpMfaBehavior",
			"  } })",
			"}",
			"$path = Join-Path (Get-Location) ('idmagic-entra-' + "+domainID+" + '-observation.json')",
			"# WriteAllText writes UTF-8 without a BOM, which IdMagic reads as JSON.",
			"[System.IO.File]::WriteAllText($path, (ConvertTo-Json -InputObject $observation -Depth 5))",
			"$path",
		),
	}
}

// entraConnect は Graph へ接続し、テナントを持つドメインでは別のテナントへの接続を止める。
func entraConnect(domainID, entraTenantID, scopes string) string {
	install := "if (-not (Get-Module -ListAvailable -Name " + graphModule + ")) { Install-Module " + graphModule + " -Scope CurrentUser }"
	check := "Get-MgDomain -DomainId " + domainID + " -ErrorAction Stop | Select-Object Id, AuthenticationType, IsVerified"
	if entraTenantID == "" {
		return lines(install,
			"Connect-MgGraph -Scopes "+scopes+" -NoWelcome",
			"# Confirm that this is the Entra tenant that owns the domain before you continue.",
			"(Get-MgContext).TenantId",
			check)
	}
	tenant := powerShellLiteral(entraTenantID)
	return lines(install,
		"Connect-MgGraph -TenantId "+tenant+" -Scopes "+scopes+" -NoWelcome",
		"if ((Get-MgContext).TenantId -ne "+tenant+") { Disconnect-MgGraph | Out-Null; throw ('Connected to an Entra tenant other than ' + "+tenant+") }",
		check)
}

func lines(parts ...string) string { return strings.Join(parts, "\n") }
