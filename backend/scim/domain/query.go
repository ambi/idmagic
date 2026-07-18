package domain

import "fmt"

// PaginationError signals that startIndex or count could not be honored per
// RFC 7644 §3.4.2.4. Callers map it to HTTP 400 with scimType "invalidValue".
type PaginationError struct {
	msg string
}

func (e *PaginationError) Error() string { return e.msg }

func newPaginationError(format string, args ...any) *PaginationError {
	return &PaginationError{msg: fmt.Sprintf(format, args...)}
}

// Page is the normalized RFC 7644 §3.4.2.4 pagination window to apply to a
// filtered result set.
type Page struct {
	// StartIndex is the 1-origin index of the first resource to return, as
	// echoed back in the ListResponse.
	StartIndex int
	// Count is the number of resources to return; 0 is valid and means
	// "return no resources, but still compute totalResults".
	Count int
}

// NormalizePage validates and clamps startIndex/count against RFC 7644
// §3.4.2.4: a startIndex below 1 is interpreted as 1, an omitted count
// defaults to maxResults, and any explicit count is clamped to maxResults.
// hasCount distinguishes an omitted count from an explicit 0.
func NormalizePage(startIndex, count *int, hasCount bool, maxResults int) (Page, error) {
	start := 1
	if startIndex != nil && *startIndex > 1 {
		start = *startIndex
	}

	c := maxResults
	if hasCount && count != nil {
		if *count < 0 {
			return Page{}, newPaginationError("count must not be negative")
		}
		c = min(*count, maxResults)
	}

	return Page{StartIndex: start, Count: c}, nil
}
