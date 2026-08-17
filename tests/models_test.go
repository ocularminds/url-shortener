package tests

import (
	"testing"
	"time"

	"github.com/ocularminds/url-shortener/core/models"
)

func TestShortLinkExpirationBoundary(t *testing.T) {
	created := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	link := models.ShortLink{Created: created, Expiry: 30}
	if link.Expired(created.AddDate(0, 0, 30).Add(-time.Nanosecond)) {
		t.Fatal("link expired before its deadline")
	}
	if !link.Expired(created.AddDate(0, 0, 30)) {
		t.Fatal("link remained active at its expiration deadline")
	}
	link.Expiry = 0
	if link.Expired(created.AddDate(10, 0, 0)) {
		t.Fatal("non-expiring link was reported expired")
	}
}
