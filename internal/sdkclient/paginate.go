package sdkclient

import "context"

// PageFetcher fetches a single page of results. Returns the items, whether
// there is a next page, and any error.
type PageFetcher[T any] func(ctx context.Context, page, perPage int) ([]T, bool, error)

// FetchAll fetches all pages up to limit using the given PageFetcher.
// If limit is 0, all pages are fetched.
func FetchAll[T any](ctx context.Context, fetch PageFetcher[T], limit int) ([]T, error) {
	perPage := 25
	if limit > 0 && limit < perPage {
		perPage = limit
	}

	var allItems []T
	page := 1

	for {
		items, hasNext, err := fetch(ctx, page, perPage)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if limit > 0 && len(allItems) >= limit {
			allItems = allItems[:limit]
			break
		}

		if !hasNext {
			break
		}
		page++
	}

	return allItems, nil
}

// StringCursorFetcher fetches a single page using string cursor-based pagination.
// Returns the items, the next cursor (empty if no more pages), and any error.
type StringCursorFetcher[T any] func(ctx context.Context, cursor string, perPage int) ([]T, string, error)

// FetchAllStringCursor fetches all pages using string cursor-based pagination.
// Used for subscriber pagination in MailerLite.
func FetchAllStringCursor[T any](ctx context.Context, fetch StringCursorFetcher[T], limit int) ([]T, error) {
	perPage := 25
	if limit > 0 && limit < perPage {
		perPage = limit
	}

	var allItems []T
	cursor := ""

	for {
		items, nextCursor, err := fetch(ctx, cursor, perPage)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if limit > 0 && len(allItems) >= limit {
			allItems = allItems[:limit]
			break
		}

		if nextCursor == "" || len(items) == 0 {
			break
		}
		cursor = nextCursor
	}

	return allItems, nil
}

// CursorFetcher fetches a single page of results using cursor-based pagination.
// Returns the items, the next cursor value (0 if no more pages), and any error.
type CursorFetcher[T any] func(ctx context.Context, after, perPage int) ([]T, int, error)

// FetchAllCursor fetches all pages using cursor-based pagination (After int).
// Used for segment subscriber pagination in MailerLite.
func FetchAllCursor[T any](ctx context.Context, fetch CursorFetcher[T], limit int) ([]T, error) {
	perPage := 25
	if limit > 0 && limit < perPage {
		perPage = limit
	}

	var allItems []T
	after := 0

	for {
		items, nextAfter, err := fetch(ctx, after, perPage)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if limit > 0 && len(allItems) >= limit {
			allItems = allItems[:limit]
			break
		}

		if nextAfter == 0 || len(items) == 0 {
			break
		}
		after = nextAfter
	}

	return allItems, nil
}
