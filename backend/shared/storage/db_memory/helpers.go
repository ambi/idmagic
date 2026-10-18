package db_memory

// DefaultScope は、保存先の所有者が指定した補完値を空のスコープに設定する。
func DefaultScope(scope *string, fallback string) {
	if *scope == "" {
		*scope = fallback
	}
}

func ScopedKey(scope, id, fallback string) string {
	DefaultScope(&scope, fallback)
	return scope + "|" + id
}
