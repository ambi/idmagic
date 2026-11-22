package domain_test

import (
	"errors"
	"testing"

	scimdomain "github.com/ambi/idmagic/backend/sourcing/internal/scim/domain"
)

// startIndex/count 省略時の既定値。 (interfaces.ListScimUsers / ListScimGroups)
func TestNormalizePageDefaults(t *testing.T) {
	page, err := scimdomain.NormalizePage(nil, nil, false, scimdomain.MaxResults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.StartIndex != 1 {
		t.Errorf("StartIndex = %d, want 1", page.StartIndex)
	}
	if page.Count != scimdomain.MaxResults {
		t.Errorf("Count = %d, want %d", page.Count, scimdomain.MaxResults)
	}
}

// startIndex < 1 は 1 に正規化する (RFC 7644 §3.4.2.4)。
func TestNormalizePageStartIndexBelowOneIsNormalized(t *testing.T) {
	page, err := scimdomain.NormalizePage(new(0), nil, false, scimdomain.MaxResults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.StartIndex != 1 {
		t.Errorf("StartIndex = %d, want 1", page.StartIndex)
	}

	page, err = scimdomain.NormalizePage(new(-5), nil, false, scimdomain.MaxResults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.StartIndex != 1 {
		t.Errorf("StartIndex = %d, want 1", page.StartIndex)
	}
}

// count=0 は「resource を返さず totalResults だけ返す」を意味し、エラーではない。
func TestNormalizePageCountZeroIsValid(t *testing.T) {
	page, err := scimdomain.NormalizePage(nil, new(0), true, scimdomain.MaxResults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Count != 0 {
		t.Errorf("Count = %d, want 0", page.Count)
	}
}

// 負の count は invalidValue (*domain.PaginationError)。
func TestNormalizePageNegativeCountIsError(t *testing.T) {
	_, err := scimdomain.NormalizePage(nil, new(-1), true, scimdomain.MaxResults)
	if err == nil {
		t.Fatal("expected error for negative count")
	}
	if _, ok := errors.AsType[*scimdomain.PaginationError](err); !ok {
		t.Fatalf("expected *domain.PaginationError, got %T: %v", err, err)
	}
}

// count が広告上限を超える場合は上限に clamp する。
func TestNormalizePageCountClampedToMax(t *testing.T) {
	page, err := scimdomain.NormalizePage(nil, new(scimdomain.MaxResults+50), true, scimdomain.MaxResults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Count != scimdomain.MaxResults {
		t.Errorf("Count = %d, want %d (clamped)", page.Count, scimdomain.MaxResults)
	}
}
