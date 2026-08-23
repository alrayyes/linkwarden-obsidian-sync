package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// fakeLinkwarden serves paginated /api/v1/search responses from an
// in-memory, newest-first list, mimicking the real API closely enough to
// exercise fetchNewLinks' pagination and stop-condition logic without a live
// instance.
func fakeLinkwarden(t *testing.T, all []link) *httptest.Server {
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

		end := cursor + pageSize
		if end > len(all) {
			end = len(all)
		}
		var page []link
		if cursor < len(all) {
			page = all[cursor:end]
		}

		resp := searchResponse{Data: page, Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
}

func linksAt(t time.Time, ids ...int) []link {
	links := make([]link, len(ids))
	for i, id := range ids {
		links[i] = link{ID: id, Name: "link", URL: "https://example.com", CreatedAt: t}
	}
	return links
}

func TestFetchNewLinksNoState(t *testing.T) {
	day1 := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	all := append(linksAt(day2, 2, 1), linksAt(day1, 0)...) // newest-first

	srv := fakeLinkwarden(t, all)
	defer srv.Close()

	c := newLinkwardenClient(srv.URL, "test-token")
	got, err := c.fetchNewLinks(time.Time{}, nil)
	if err != nil {
		t.Fatalf("fetchNewLinks() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("fetchNewLinks() got %d links, want 3", len(got))
	}
}

func TestFetchNewLinksStopsAtLastSynced(t *testing.T) {
	day1 := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)

	all := append(linksAt(day2, 2, 1), linksAt(day1, 0)...)

	srv := fakeLinkwarden(t, all)
	defer srv.Close()

	c := newLinkwardenClient(srv.URL, "test-token")
	got, err := c.fetchNewLinks(day1, map[int]bool{0: true})
	if err != nil {
		t.Fatalf("fetchNewLinks() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("fetchNewLinks() got %d links, want 2 (only day2's)", len(got))
	}
}

func TestFetchNewLinksSameTimestampBoundary(t *testing.T) {
	same := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	all := linksAt(same, 3, 2, 1) // three links saved in the same second

	srv := fakeLinkwarden(t, all)
	defer srv.Close()

	c := newLinkwardenClient(srv.URL, "test-token")
	// Already synced link 1 and 2 at this exact timestamp; only 3 is new.
	got, err := c.fetchNewLinks(same, map[int]bool{1: true, 2: true})
	if err != nil {
		t.Fatalf("fetchNewLinks() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("fetchNewLinks() = %+v, want only link 3", got)
	}
}

func TestFetchNewLinksUnauthorized(t *testing.T) {
	srv := fakeLinkwarden(t, nil)
	defer srv.Close()

	c := newLinkwardenClient(srv.URL, "wrong-token")
	if _, err := c.fetchNewLinks(time.Time{}, nil); err == nil {
		t.Fatal("fetchNewLinks() with a bad token: want an error, got nil")
	}
}
