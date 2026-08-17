// Package models contains domain entities with no infrastructure dependencies.
package models

import "time"

// ShortLink represents a stored redirect.
type ShortLink struct {
	Shortened string    `json:"slug"`
	Original  string    `json:"url"`
	Expiry    int       `json:"expiryDays"`
	Created   time.Time `json:"createdAt"`
	Hits      uint64    `json:"hits"`
}

func (link ShortLink) Expired(now time.Time) bool {
	return link.Expiry > 0 && !now.Before(link.Created.AddDate(0, 0, link.Expiry))
}
