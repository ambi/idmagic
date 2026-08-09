package db_memory

import "testing"

type keysetItem struct {
	primary string
	id      string
}

func keysetKeyOf(i keysetItem) (string, string) { return i.primary, i.id }

func TestKeysetPageFirstPageAscending(t *testing.T) {
	items := []keysetItem{{"charlie", "3"}, {"alpha", "1"}, {"bravo", "2"}}
	page := KeysetPage(items, keysetKeyOf, false, "", "", 2)
	if len(page) != 2 || page[0].primary != "alpha" || page[1].primary != "bravo" {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestKeysetPageContinuationAscending(t *testing.T) {
	items := []keysetItem{{"charlie", "3"}, {"alpha", "1"}, {"bravo", "2"}, {"delta", "4"}}
	page := KeysetPage(items, keysetKeyOf, false, "bravo", "2", 2)
	if len(page) != 2 || page[0].primary != "charlie" || page[1].primary != "delta" {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestKeysetPageDescending(t *testing.T) {
	items := []keysetItem{{"alpha", "1"}, {"charlie", "3"}, {"bravo", "2"}}
	page := KeysetPage(items, keysetKeyOf, true, "", "", 2)
	if len(page) != 2 || page[0].primary != "charlie" || page[1].primary != "bravo" {
		t.Fatalf("unexpected descending page: %+v", page)
	}
}

func TestKeysetPageTiesBrokenByID(t *testing.T) {
	items := []keysetItem{{"same", "3"}, {"same", "1"}, {"same", "2"}}
	page := KeysetPage(items, keysetKeyOf, false, "", "", 10)
	if len(page) != 3 || page[0].id != "1" || page[1].id != "2" || page[2].id != "3" {
		t.Fatalf("unexpected tie-break order: %+v", page)
	}
}

func TestKeysetPageZeroLimitReturnsAll(t *testing.T) {
	items := []keysetItem{{"b", "2"}, {"a", "1"}}
	page := KeysetPage(items, keysetKeyOf, false, "", "", 0)
	if len(page) != 2 {
		t.Fatalf("expected limit<=0 to mean unbounded, got %d items", len(page))
	}
}

func TestKeysetPageBeforeAscendingReturnsNearestPageInCanonicalOrder(t *testing.T) {
	items := []keysetItem{{"delta", "4"}, {"alpha", "1"}, {"charlie", "3"}, {"bravo", "2"}}
	page := KeysetPageBefore(items, keysetKeyOf, false, "delta", "4", 2)
	if len(page) != 2 || page[0].primary != "bravo" || page[1].primary != "charlie" {
		t.Fatalf("unexpected previous page: %+v", page)
	}
}

func TestKeysetPageBeforeDescendingReturnsNearestPageInCanonicalOrder(t *testing.T) {
	items := []keysetItem{{"alpha", "1"}, {"delta", "4"}, {"bravo", "2"}, {"charlie", "3"}}
	page := KeysetPageBefore(items, keysetKeyOf, true, "bravo", "2", 2)
	if len(page) != 2 || page[0].primary != "delta" || page[1].primary != "charlie" {
		t.Fatalf("unexpected descending previous page: %+v", page)
	}
}

func TestKeysetPageBeforeKeepsIDTieBreakOrder(t *testing.T) {
	items := []keysetItem{{"same", "4"}, {"same", "1"}, {"same", "3"}, {"same", "2"}}
	page := KeysetPageBefore(items, keysetKeyOf, false, "same", "4", 2)
	if len(page) != 2 || page[0].id != "2" || page[1].id != "3" {
		t.Fatalf("unexpected tie-break previous page: %+v", page)
	}
}

func TestKeysetPagesRemainAddressableAcrossInsertAndDelete(t *testing.T) {
	items := []keysetItem{{"alpha", "1"}, {"bravo", "2"}, {"charlie", "3"}, {"delta", "4"}, {"echo", "5"}}
	first := KeysetPage(items, keysetKeyOf, false, "", "", 2)
	second := KeysetPage(items, keysetKeyOf, false, first[1].primary, first[1].id, 2)

	// An insertion before the current boundary and deletion after it must not
	// change which existing rows are immediately before the second page.
	mutated := []keysetItem{{"aardvark", "0"}, {"alpha", "1"}, {"bravo", "2"}, {"charlie", "3"}, {"delta", "4"}}
	previous := KeysetPageBefore(mutated, keysetKeyOf, false, second[0].primary, second[0].id, 2)
	if len(previous) != 2 || previous[0].primary != "alpha" || previous[1].primary != "bravo" {
		t.Fatalf("unexpected page after concurrent mutation: %+v", previous)
	}
}

func TestKeysetPageBeforeFirstBoundaryIsEmpty(t *testing.T) {
	items := []keysetItem{{"alpha", "1"}, {"bravo", "2"}}
	page := KeysetPageBefore(items, keysetKeyOf, false, "alpha", "1", 2)
	if len(page) != 0 {
		t.Fatalf("expected no page before the first boundary, got %+v", page)
	}
}
