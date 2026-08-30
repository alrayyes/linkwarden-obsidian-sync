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

// fakeSearchResponse mirrors the shape of Linkwarden's own /api/v1/search
// response, kept independent of the client's internal decode type so the
// fake exercises the real API contract rather than the implementation.
type fakeSearchResponse struct {
	Data    []linkwarden.Link `json:"data"`
	Success bool              `json:"success"`
}

// fakeLinkwarden serves paginated /api/v1/search responses from an
// in-memory, newest-first list, mimicking the real API closely enough to
// exercise FetchNewLinks' pagination and stop-condition logic without a live
// instance.
func fakeLinkwarden(t *testing.T, all []linkwarden.Link) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		cursor := 0
		if c := r.URL.Query().Get("cursor"); c != "" {
			n, err := strconv.Atoi(c)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)

				return
			}
			cursor = n
		}

		end := min(cursor+linkwarden.PageSize, len(all))
		var page []linkwarden.Link
		if cursor < len(all) {
			page = all[cursor:end]
		}

		resp := fakeSearchResponse{Data: page, Success: true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
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
		srv := fakeLinkwarden(t, all)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "test-token")
		got, err := c.FetchNewLinks(time.Time{}, nil)
		require.NoError(t, err)
		require.Len(t, got, 3)
	})

	t.Run("stops at last synced", func(t *testing.T) {
		t.Parallel()

		all := append(linksAt(day2, 2, 1), linksAt(day1, 0)...)
		srv := fakeLinkwarden(t, all)
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
		srv := fakeLinkwarden(t, all)
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

		srv := fakeLinkwarden(t, nil)
		defer srv.Close()

		c := linkwarden.NewClient(srv.URL, "wrong-token")
		_, err := c.FetchNewLinks(time.Time{}, nil)
		require.Error(t, err)
	})
}
