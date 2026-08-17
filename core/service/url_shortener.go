package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/ocularminds/url-shortener/core/models"
	"github.com/ocularminds/url-shortener/core/repository"
)

const (
	defaultExpiryDays = 30
	maxSlugAttempts   = 5
)

var (
	ErrNotFound = errors.New("short link not found")
	validSlug   = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)
)

type Option func(*URLShortener)

func WithClock(clock func() time.Time) Option {
	return func(shortener *URLShortener) {
		if clock != nil {
			shortener.now = clock
		}
	}
}

type URLShortener struct {
	repository repository.LinkRepository
	generator  SlugGenerator
	validator  URLValidator
	now        func() time.Time
	expiryDays int
}

func NewURLShortener(repository repository.LinkRepository, generator SlugGenerator, options ...Option) (*URLShortener, error) {
	if repository == nil || generator == nil {
		return nil, errors.New("repository and slug generator are required")
	}
	shortener := &URLShortener{
		repository: repository,
		generator:  generator,
		validator:  NewURLValidator(),
		now:        time.Now,
		expiryDays: defaultExpiryDays,
	}
	for _, option := range options {
		option(shortener)
	}
	return shortener, nil
}

func (shortener *URLShortener) Create(ctx context.Context, original string) (models.ShortLink, bool, error) {
	if err := shortener.validator.Validate(original); err != nil {
		return models.ShortLink{}, false, err
	}

	existing, err := shortener.repository.FindByOriginal(ctx, original)
	if err == nil && !existing.Expired(shortener.now()) {
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return models.ShortLink{}, false, fmt.Errorf("find original URL: %w", err)
	}

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug, err := shortener.generator.Generate()
		if err != nil {
			return models.ShortLink{}, false, fmt.Errorf("generate secure slug: %w", err)
		}
		if !validSlug.MatchString(slug) {
			return models.ShortLink{}, false, errors.New("slug generator returned an invalid value")
		}
		link := models.ShortLink{
			Shortened: slug,
			Original:  original,
			Expiry:    shortener.expiryDays,
			Created:   shortener.now().UTC(),
		}
		if err := shortener.repository.Create(ctx, link); errors.Is(err, repository.ErrConflict) {
			continue
		} else if err != nil {
			return models.ShortLink{}, false, err
		}
		return link, true, nil
	}
	return models.ShortLink{}, false, errors.New("could not allocate a unique slug")
}

func (shortener *URLShortener) Resolve(ctx context.Context, slug string) (models.ShortLink, error) {
	if !validSlug.MatchString(slug) {
		return models.ShortLink{}, ErrNotFound
	}
	link, err := shortener.repository.FindBySlug(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return models.ShortLink{}, ErrNotFound
	}
	if err != nil {
		return models.ShortLink{}, err
	}
	if link.Expired(shortener.now()) {
		return models.ShortLink{}, ErrNotFound
	}
	if err := shortener.repository.IncrementHits(ctx, slug); err != nil {
		return models.ShortLink{}, err
	}
	link.Hits++
	return link, nil
}
