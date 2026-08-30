// Package linkwarden is a client for the Linkwarden bookmarking API.
package linkwarden

import (
	"encoding/json"
	"errors"
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

// errUnexpectedStatus is returned when Linkwarden's HTTP response status
// isn't 200 OK.
var errUnexpectedStatus = errors.New("unexpected status from linkwarden")

// errLinkwardenFailure is returned when Linkwarden's own response body
// reports success: false.
var errLinkwardenFailure = errors.New("linkwarden reported failure")

// Link is a single saved bookmark, as returned by Linkwarden's search API.
type Link struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Tags        []Tag      `json:"tags"`
	Collection  Collection `json:"collection"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// Tag is a label attached to a Link.
type Tag struct {
	Name string `json:"name"`
}

// Collection is the Linkwarden collection a Link belongs to.
type Collection struct {
	Name string `json:"name"`
}

type searchResponse struct {
	Data    []Link `json:"data"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Client is a Linkwarden API client, scoped to one base URL and access token.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient builds a Client for the Linkwarden instance at baseURL,
// authenticating with token.
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

		var stop bool
		fresh, stop = appendFresh(fresh, page, since, seenAtSince)
		if stop || len(page) < PageSize {
			return fresh, nil
		}
		cursor += PageSize
	}
}

// appendFresh appends the links in page that are newer than since (or, at
// the exact boundary, not already recorded in seenAtSince) onto fresh. It
// reports whether page reached a link already synced, meaning the caller
// should stop paging.
func appendFresh(fresh, page []Link, since time.Time, seenAtSince map[int]bool) (result []Link, stop bool) {
	for _, l := range page {
		if l.CreatedAt.Before(since) {
			return fresh, true
		}
		if l.CreatedAt.Equal(since) && seenAtSince[l.ID] {
			continue
		}
		fresh = append(fresh, l)
	}

	return fresh, false
}

func (c *Client) fetchPage(cursor int) ([]Link, error) {
	url := fmt.Sprintf("%s/api/v1/search?sort=%d&cursor=%d", c.baseURL, sortDateNewestFirst, cursor)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s for %s", errUnexpectedStatus, resp.Status, url)
	}

	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding linkwarden response: %w", err)
	}
	if !parsed.Success {
		return nil, fmt.Errorf("%w: %s", errLinkwardenFailure, parsed.Message)
	}

	return parsed.Data, nil
}
