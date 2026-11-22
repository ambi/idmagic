package usecases

import scimdomain "github.com/ambi/idmagic/backend/sourcing/internal/scim/domain"

// paginate applies RFC 7644 §3.4.2.4 startIndex/count semantics to an
// already-filtered result set. Shared by ListUsers and ListGroups.
func paginate(items []map[string]any, query ListQuery) (ListResult, error) {
	page, err := scimdomain.NormalizePage(query.StartIndex, query.Count, query.HasCount, scimdomain.MaxResults)
	if err != nil {
		return ListResult{}, err
	}

	total := len(items)
	begin := min(page.StartIndex-1, total)
	end := min(begin+page.Count, total)
	pageItems := items[begin:end]

	return ListResult{
		Total:        total,
		Items:        pageItems,
		StartIndex:   page.StartIndex,
		ItemsPerPage: len(pageItems),
	}, nil
}
