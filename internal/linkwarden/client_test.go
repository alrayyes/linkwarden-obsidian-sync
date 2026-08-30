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
// in-memory, newest-first list, mimicking the real API's Prisma cursor
// pagination: an omitted cursor starts from the beginning, a given cursor
// is the ID of the last link the caller already has, and nextCursor is the
// last link's ID in the response (or nil once a page comes back shorter
// than pageSize) — not a numeric offset. requests, if non-nil, is
// incremented once per request received.
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

// cursorStart finds where in all (newest-first) the page after cursor
// begins — the position just past the link whose ID matches cursor, or 0
// when cursor is empty (the first page). ok is false only on a
// non-numeric cursor, mirroring the real API's 400 on a malformed one.
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

func linksAt(t time.Time, ids ...int) []linkwarden.Link {
	links := make([]linkwarden.Link, len(ids))
	for i, id := range ids {
		links[i] = linkwarden.Link{ID: id, Name: "link", URL: "https://example.com", CreatedAt: t}
	}

	return links
}

func TestFetchNewLinks(t *testing.T) {
	t.Parallel()

	day1 := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	t.Run("no state", func(t *testing.T) {
		t.Parallel()

		all := append(linksAt(day2, 2, 1), linksAt(day1, 0)...) // newest-first
		srv := fakeLinkwarden(t, all, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchNewLinks(time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, got, 3)
	})

	t.Run("stops at last synced", func(t *testing.T) {
		t.Parallel()

		all := append(linksAt(day2, 2, 1), linksAt(day1, 0)...)
		srv := fakeLinkwarden(t, all, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchNewLinks(day1, map[int]bool{0: true})
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("same timestamp boundary", func(t *testing.T) {
		t.Parallel()

		same := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
		all := linksAt(same, 3, 2, 1) // three links saved in the same second
		srv := fakeLinkwarden(t, all, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		// Already synced link 1 and 2 at this exact timestamp; only 3 is new.
		got, err := c.FetchNewLinks(same, map[int]bool{1: true, 2: true})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, 3, got[0].ID)
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		srv := fakeLinkwarden(t, nil, 50, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "wrong-token")
		_, err := c.FetchNewLinks(time.Time{}, nil)
		require.Error(t, err)
	})
}

// TestFetchNewLinksPagination covers the multi-page cursor mechanics
// specifically — split from TestFetchNewLinks so neither function grows
// past this repo's funlen limit.
func TestFetchNewLinksPagination(t *testing.T) {
	t.Parallel()

	day1 := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	t.Run("pages using the server's ID-based nextCursor, not an offset", func(t *testing.T) {
		t.Parallel()

		// Small page size forces multiple requests; IDs are deliberately not
		// contiguous with position, to catch a client that assumes an offset
		// instead of echoing the server's own nextCursor back.
		all := linksAt(day1, 50, 40, 30, 20, 10)
		srv := fakeLinkwarden(t, all, 2, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchNewLinks(time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, got, 5)
	})

	t.Run("stops after a short page with no further request", func(t *testing.T) {
		t.Parallel()

		var requests int
		all := linksAt(day1, 2, 1)
		srv := fakeLinkwarden(t, all, 50, &requests)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchNewLinks(time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, 1, requests)
	})
}
