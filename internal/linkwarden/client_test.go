package linkwarden_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/stretchr/testify/require"
)

// fakeSearchData and fakeSearchResponse mirror the actual shape of
// Linkwarden's own /api/v1/search response (checked against
// apps/web/lib/api/controllers/search/searchLinks.ts and
// apps/web/pages/api/v1/search/index.ts on linkwarden/linkwarden) — links
// live under data.links, not a bare data array — kept independent of the
// client's internal decode type so the fake exercises the real API
// contract rather than the implementation.
type fakeSearchData struct {
	Links      []linkwarden.Link `json:"links"`
	NextCursor *int              `json:"nextCursor"`
}

type fakeSearchResponse struct {
	Data    fakeSearchData `json:"data"`
	Success bool           `json:"success"`
}

// fakeLinkwarden serves paginated /api/v1/search responses from an
// in-memory list, mimicking the real API's Prisma cursor pagination: an
// omitted cursor starts from the beginning, a given cursor is the ID of
// the last link the caller already has, and nextCursor is the last link's
// ID in the response (or nil once a page comes back shorter than
// pageSize) — not a numeric offset. requests, if non-nil, is incremented
// once per request received.
func fakeLinkwarden(t *testing.T, all []linkwarden.Link, pageSize int, requests *int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests != nil {
			*requests++
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		start, ok := cursorStart(all, r.URL.Query().Get("cursor"))
		if !ok {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		end := min(start+pageSize, len(all))
		var page []linkwarden.Link
		if start < len(all) {
			page = all[start:end]
		}

		var nextCursor *int
		if len(page) == pageSize {
			id := page[len(page)-1].ID
			nextCursor = &id
		}

		resp := fakeSearchResponse{Data: fakeSearchData{Links: page, NextCursor: nextCursor}, Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck,gosec // response goes to httptest's in-memory ResponseRecorder, not a real client
	}))
}

// cursorStart finds where in all the page after cursor begins — the
// position just past the link whose ID matches cursor, or 0 when cursor
// is empty (the first page). ok is false only on a non-numeric cursor,
// mirroring the real API's 400 on a malformed one.
func cursorStart(all []linkwarden.Link, cursor string) (start int, ok bool) {
	if cursor == "" {
		return 0, true
	}

	cursorID, err := strconv.Atoi(cursor)
	if err != nil {
		return 0, false
	}

	for i, l := range all {
		if l.ID == cursorID {
			return i + 1, true
		}
	}

	return len(all), true
}

func linksWithIDs(ids ...int) []linkwarden.Link {
	links := make([]linkwarden.Link, len(ids))
	for i, id := range ids {
		links[i] = linkwarden.Link{ID: id, Name: "link", URL: "https://example.com", CreatedAt: time.Now()}
	}

	return links
}

func TestFetchAllLinks(t *testing.T) {
	t.Parallel()

	t.Run("empty collection", func(t *testing.T) {
		t.Parallel()

		srv := fakeLinkwarden(t, nil, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchAllLinks()
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("single page", func(t *testing.T) {
		t.Parallel()

		all := linksWithIDs(3, 2, 1)
		srv := fakeLinkwarden(t, all, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchAllLinks()
		require.NoError(t, err)
		require.Len(t, got, 3)
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		srv := fakeLinkwarden(t, nil, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "wrong-token")
		_, err := c.FetchAllLinks()
		require.Error(t, err)
	})
}

// TestFetchAllLinksPagination covers the multi-page cursor mechanics
// specifically — split from TestFetchAllLinks so neither function grows
// past this repo's funlen limit.
func TestFetchAllLinksPagination(t *testing.T) {
	t.Parallel()

	t.Run("pages using the server's ID-based nextCursor, not an offset", func(t *testing.T) {
		t.Parallel()

		// Small page size forces multiple requests; IDs are deliberately not
		// contiguous with position, to catch a client that assumes an offset
		// instead of echoing the server's own nextCursor back.
		all := linksWithIDs(50, 40, 30, 20, 10)
		srv := fakeLinkwarden(t, all, 2, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchAllLinks()
		require.NoError(t, err)
		require.Len(t, got, 5)
	})

	t.Run("stops after a short page with no further request", func(t *testing.T) {
		t.Parallel()

		var requests int
		all := linksWithIDs(2, 1)
		srv := fakeLinkwarden(t, all, 50, &requests)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchAllLinks()
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, 1, requests)
	})

	t.Run("collection exactly one page long still resolves in two requests", func(t *testing.T) {
		t.Parallel()

		var requests int
		all := linksWithIDs(2, 1) // pageSize below is also 2
		srv := fakeLinkwarden(t, all, 2, &requests)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchAllLinks()
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, 2, requests, "a full page always implies a nextCursor, so one more (empty) request confirms the end")
	})
}
