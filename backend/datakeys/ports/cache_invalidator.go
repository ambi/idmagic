package ports

// CacheInvalidator lets lifecycle usecases invalidate a cached unwrapped DEK
// without depending on a concrete cache type. Implementations must be safe
// for concurrent worker replicas (ADR-148: no replica keeps encrypting with a
// DEK that is no longer active after rotate/disable/destroy).
type CacheInvalidator interface {
	Invalidate(tenantID string)
}
