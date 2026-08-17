// Package repository defines persistence contracts and implementations.
package repository

import (
	"context"
	"errors"

	"github.com/ocularminds/url-shortener/core/models"
)

var (
	ErrNotFound = errors.New("short link not found")
	ErrConflict = errors.New("short link already exists")
)

type LinkRepository interface {
	FindBySlug(context.Context, string) (models.ShortLink, error)
	FindByOriginal(context.Context, string) (models.ShortLink, error)
	Create(context.Context, models.ShortLink) error
	IncrementHits(context.Context, string) error
}
