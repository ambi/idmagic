//go:build !darwin

package testing_postgres

// sysvUnavailable は macOS のサンドボックスだけが持つ問題への対処であり、
// ほかの OS では検査しない。
func sysvUnavailable() error { return nil }
