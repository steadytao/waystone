// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"net/url"
	"strconv"
)

func paginate[T any](ctx context.Context, client *Client, path string, query url.Values, handle func([]T) error) error {
	if query == nil {
		query = url.Values{}
	}
	query.Set("per_page", "100")

	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		var items []T
		if err := client.get(ctx, path, query, &items); err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		if err := handle(items); err != nil {
			return err
		}
		if len(items) < 100 {
			return nil
		}
	}
}
