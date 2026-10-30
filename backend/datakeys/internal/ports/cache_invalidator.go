package ports

// CacheInvalidator lets lifecycle usecases drop this process's cached unwrapped
// DEKs without depending on a concrete cache type. It reaches only this process:
// the cache itself must not rely on it to stop using a version.
type CacheInvalidator interface {
	Invalidate(tenantID string)
}
