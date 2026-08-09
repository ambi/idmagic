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
