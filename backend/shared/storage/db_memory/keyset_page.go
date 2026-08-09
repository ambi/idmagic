package db_memory

import (
	"slices"
	"strings"
)

// KeysetPage sorts items by a two-part (primary, id) key and returns up to
// limit items strictly after (afterPrimary, afterID) — "", "" for the first
// page, limit<=0 for unbounded. Shared by every ports.*Repository.ListPage
// in-memory implementation (wi-159, ADR-158) so each only supplies its own
// key extraction, not the sort/seek/limit mechanics. Sorts items in place
// (like slices.SortFunc); callers pass a slice already private to the call
// (e.g. freshly built via append while iterating a map).
func KeysetPage[T any](items []T, key func(T) (primary, id string), desc bool, afterPrimary, afterID string, limit int) []T {
	order := 1
	if desc {
		order = -1
	}
	cmpKey := func(p1, id1, p2, id2 string) int {
		if c := strings.Compare(p1, p2); c != 0 {
			return c * order
		}
		return strings.Compare(id1, id2) * order
	}
	slices.SortFunc(items, func(a, b T) int {
		ap, aid := key(a)
		bp, bid := key(b)
		return cmpKey(ap, aid, bp, bid)
	})
	if afterPrimary != "" || afterID != "" {
		idx, found := slices.BinarySearchFunc(items, [2]string{afterPrimary, afterID}, func(item T, k [2]string) int {
			p, id := key(item)
			return cmpKey(p, id, k[0], k[1])
		})
		start := idx
		if found {
			start = idx + 1
		}
		items = items[start:]
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// KeysetPageBefore returns the nearest page strictly before the supplied
// boundary. The result remains in the same canonical order as KeysetPage even
// though a database adapter normally obtains it through a reverse scan.
func KeysetPageBefore[T any](items []T, key func(T) (primary, id string), desc bool, beforePrimary, beforeID string, limit int) []T {
	order := 1
	if desc {
		order = -1
	}
	cmpKey := func(p1, id1, p2, id2 string) int {
		if c := strings.Compare(p1, p2); c != 0 {
			return c * order
		}
		return strings.Compare(id1, id2) * order
	}
	slices.SortFunc(items, func(a, b T) int {
		ap, aid := key(a)
		bp, bid := key(b)
		return cmpKey(ap, aid, bp, bid)
	})
	if beforePrimary != "" || beforeID != "" {
		idx, _ := slices.BinarySearchFunc(items, [2]string{beforePrimary, beforeID}, func(item T, k [2]string) int {
			p, id := key(item)
			return cmpKey(p, id, k[0], k[1])
		})
		items = items[:idx]
	}
	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items
}
