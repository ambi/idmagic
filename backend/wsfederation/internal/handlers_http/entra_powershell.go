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
