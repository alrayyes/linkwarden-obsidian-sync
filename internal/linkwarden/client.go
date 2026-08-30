// Package linkwarden is a client for the Linkwarden bookmarking API.
package linkwarden

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// sortDateNewestFirst matches Linkwarden's Sort enum (packages/types/global.ts):
// DateNewestFirst = 0. Fetching newest-first lets a paginated scan stop at the
// first link already seen, instead of paging through the whole collection
// every run.
const sortDateNewestFirst = 0

// PageSize matches Linkwarden's own default PAGINATION_TAKE_COUNT. The API
// doesn't report it back, so a page shorter than this is the signal we've
// reached the end.
const PageSize = 50

type Link struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Tags        []Tag      `json:"tags"`
	Collection  Collection `json:"collection"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Tag struct {
	Name string `json:"name"`
}

type Collection struct {
	Name string `json:"name"`
}

type searchResponse struct {
	Data    []Link `json:"data"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchNewLinks pages through /api/v1/search, newest first, stopping the
// moment it sees a link that's already been synced (createdAt at or before
// `since`, and — for the boundary where several links share the exact same
// timestamp — an ID already recorded in `seenAtSince`). Returned links are in
// newest-first order; the caller reverses them before writing so notes land
// in the vault in the order they were actually saved.
func (c *Client) FetchNewLinks(since time.Time, seenAtSince map[int]bool) ([]Link, error) {
	var fresh []Link
	cursor := 0

	for {
		page, err := c.fetchPage(cursor)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return fresh, nil
		}

		for _, l := range page {
			if l.CreatedAt.Before(since) {
				return fresh, nil
			}
			if l.CreatedAt.Equal(since) && seenAtSince[l.ID] {
				continue
			}
			fresh = append(fresh, l)
		}

		if len(page) < PageSize {
			return fresh, nil
		}
		cursor += PageSize
	}
}

func (c *Client) fetchPage(cursor int) ([]Link, error) {
	url := fmt.Sprintf("%s/api/v1/search?sort=%d&cursor=%d", c.baseURL, sortDateNewestFirst, cursor)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("linkwarden returned %s for %s", resp.Status, url)
	}

	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding linkwarden response: %w", err)
	}
	if !parsed.Success {
		return nil, fmt.Errorf("linkwarden reported failure: %s", parsed.Message)
	}

	return parsed.Data, nil
}
