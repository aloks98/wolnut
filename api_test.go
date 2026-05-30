package main

import (
	"testing"
	"time"
)

func TestLatestReleaseCacheSnapshot(t *testing.T) {
	c := newLatestReleaseCache(time.Hour)

	// Cold cache: nothing fresh, nothing stale, no backoff.
	if s := c.snapshot(); s.fresh || s.stale || s.backoff {
		t.Errorf("cold snapshot = %+v, want all false", s)
	}

	// After a success, value is both fresh and stale, no backoff.
	c.setSuccess("1.2.3", "https://example.com/r")
	s := c.snapshot()
	if !s.fresh || !s.stale || s.backoff || s.version != "1.2.3" {
		t.Errorf("post-success snapshot = %+v, want fresh+stale, version 1.2.3", s)
	}

	// Age the success past ttl: stale but no longer fresh.
	c.mu.Lock()
	c.fetchedAt = time.Now().Add(-2 * time.Hour)
	c.mu.Unlock()
	if s := c.snapshot(); s.fresh || !s.stale {
		t.Errorf("aged snapshot = %+v, want stale and not fresh", s)
	}

	// A failure starts the backoff window while keeping the stale value.
	c.setError()
	if s := c.snapshot(); !s.backoff || !s.stale {
		t.Errorf("post-error snapshot = %+v, want backoff and stale", s)
	}

	// A later success clears the backoff window and refreshes.
	c.setSuccess("1.3.0", "https://example.com/r2")
	if s := c.snapshot(); !s.fresh || s.backoff || s.version != "1.3.0" {
		t.Errorf("recovered snapshot = %+v, want fresh, no backoff, version 1.3.0", s)
	}
}

func TestLatestReleaseCacheBackoffExpiry(t *testing.T) {
	c := newLatestReleaseCache(time.Hour)
	c.setError()
	if s := c.snapshot(); !s.backoff {
		t.Fatal("expected backoff immediately after error")
	}
	// Push the failure past errTTL: backoff window has elapsed.
	c.mu.Lock()
	c.lastErrAt = time.Now().Add(-c.errTTL - time.Minute)
	c.mu.Unlock()
	if s := c.snapshot(); s.backoff {
		t.Error("expected backoff to expire after errTTL")
	}
}
