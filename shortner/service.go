package shortner

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
)

const maxSlugAttempts = 5

var validSlug = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)

type LinkService interface {
	Create(context.Context, string) (ShortLink, bool, error)
	Resolve(context.Context, string) (ShortLink, error)
}

type URLShortener struct {
	repository LinkRepository
	generator  SlugGenerator
	validator  URLValidator
	now        func() time.Time
	expiryDays int
}

func NewURLShortener(repository LinkRepository, generator SlugGenerator) (*URLShortener, error) {
	if repository == nil || generator == nil {
		return nil, errors.New("repository and slug generator are required")
	}
	return &URLShortener{
		repository: repository,
		generator:  generator,
		validator:  NewURLValidator(),
		now:        time.Now,
		expiryDays: DefaultExpiryDays,
	}, nil
}

func (service *URLShortener) Create(ctx context.Context, original string) (ShortLink, bool, error) {
	if err := service.validator.Validate(original); err != nil {
		return ShortLink{}, false, err
	}

	existing, err := service.repository.FindByOriginal(ctx, original)
	if err == nil && !existing.Expired(service.now()) {
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return ShortLink{}, false, fmt.Errorf("find original URL: %w", err)
	}

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug, err := service.generator.Generate()
		if err != nil {
			return ShortLink{}, false, fmt.Errorf("generate secure slug: %w", err)
		}
		link := ShortLink{
			Shortened: slug,
			Original:  original,
			Expiry:    service.expiryDays,
			Created:   service.now().UTC(),
		}
		if err := service.repository.Create(ctx, link); errors.Is(err, ErrConflict) {
			continue
		} else if err != nil {
			return ShortLink{}, false, err
		}
		return link, true, nil
	}
	return ShortLink{}, false, errors.New("could not allocate a unique slug")
}

func (service *URLShortener) Resolve(ctx context.Context, slug string) (ShortLink, error) {
	if !validSlug.MatchString(slug) {
		return ShortLink{}, ErrNotFound
	}
	link, err := service.repository.FindBySlug(ctx, slug)
	if err != nil {
		return ShortLink{}, err
	}
	if link.Expired(service.now()) {
		return ShortLink{}, ErrNotFound
	}
	if err := service.repository.IncrementHits(ctx, slug); err != nil {
		return ShortLink{}, err
	}
	link.Hits++
	return link, nil
}
