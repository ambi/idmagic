package domain

import "errors"

// SoftQuotaUsage は各保存先を読んだ時点の測定値であり、全体の同時点スナップショットではない。
type SoftQuotaUsage struct {
	AuditEventsRetained  int
	ExportArtifactsBytes int
}

func (u SoftQuotaUsage) Validate() error {
	if u.AuditEventsRetained < 0 || u.ExportArtifactsBytes < 0 {
		return errors.New("soft quota: negative usage")
	}
	return nil
}

func IsSoftQuota(resource string) bool {
	return resource == ResourceAuditEventsRetained || resource == ResourceExportArtifactsBytes
}

// ExceededResources の順序は、監査件数、成果物バイト数の順に固定する。
func (u SoftQuotaUsage) ExceededResources(quota *TenantQuota) []string {
	var resources []string
	if u.AuditEventsRetained > quota.EffectiveLimit(ResourceAuditEventsRetained) {
		resources = append(resources, ResourceAuditEventsRetained)
	}
	if u.ExportArtifactsBytes > quota.EffectiveLimit(ResourceExportArtifactsBytes) {
		resources = append(resources, ResourceExportArtifactsBytes)
	}
	return resources
}
