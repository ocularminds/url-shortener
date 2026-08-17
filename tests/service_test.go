package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ocularminds/url-shortener/core/models"
	"github.com/ocularminds/url-shortener/core/repository"
	"github.com/ocularminds/url-shortener/core/service"
)

var fixedTime = time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

type memoryRepository struct {
	mu      sync.Mutex
	bySlug  map[string]models.ShortLink
	created int
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{bySlug: make(map[string]models.ShortLink)}
}

func (store *memoryRepository) FindBySlug(_ context.Context, slug string) (models.ShortLink, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	link, found := store.bySlug[slug]
	if !found {
		return models.ShortLink{}, repository.ErrNotFound
	}
	return link, nil
}

func (store *memoryRepository) FindByOriginal(_ context.Context, original string) (models.ShortLink, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, link := range store.bySlug {
		if link.Original == original {
			return link, nil
		}
	}
	return models.ShortLink{}, repository.ErrNotFound
}

func (store *memoryRepository) Create(_ context.Context, link models.ShortLink) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, found := store.bySlug[link.Shortened]; found {
		return repository.ErrConflict
	}
	store.bySlug[link.Shortened] = link
	store.created++
	return nil
}

func (store *memoryRepository) IncrementHits(_ context.Context, slug string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	link, found := store.bySlug[slug]
	if !found {
		return repository.ErrNotFound
	}
	link.Hits++
	store.bySlug[slug] = link
	return nil
}

type sequenceGenerator struct {
	mu     sync.Mutex
	values []string
	error  error
}

func (generator *sequenceGenerator) Generate() (string, error) {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	if generator.error != nil {
		return "", generator.error
	}
	value := generator.values[0]
	generator.values = generator.values[1:]
	return value, nil
}

func newTestService(t *testing.T, store repository.LinkRepository, slugs ...string) *service.URLShortener {
	t.Helper()
	shortener, err := service.NewURLShortener(
		store,
		&sequenceGenerator{values: slugs},
		service.WithClock(func() time.Time { return fixedTime }),
	)
	if err != nil {
		t.Fatal(err)
	}
	return shortener
}

func TestURLShortenerCreatesAndReusesLink(t *testing.T) {
	store := newMemoryRepository()
	shortener := newTestService(t, store, "AbCd1234")

	created, isNew, err := shortener.Create(context.Background(), "https://example.com/long")
	if err != nil || !isNew {
		t.Fatalf("first Create() = (%+v, %v, %v), want a new link", created, isNew, err)
	}
	existing, isNew, err := shortener.Create(context.Background(), "https://example.com/long")
	if err != nil || isNew || existing.Shortened != created.Shortened {
		t.Fatalf("second Create() = (%+v, %v, %v), want existing link", existing, isNew, err)
	}
	if store.created != 1 {
		t.Fatalf("repository created %d records, want 1", store.created)
	}
}

func TestURLShortenerRetriesSlugCollision(t *testing.T) {
	store := newMemoryRepository()
	store.bySlug["Collide1"] = models.ShortLink{Shortened: "Collide1", Original: "https://other.example"}
	shortener := newTestService(t, store, "Collide1", "Unique12")

	link, isNew, err := shortener.Create(context.Background(), "https://example.com/long")
	if err != nil || !isNew || link.Shortened != "Unique12" {
		t.Fatalf("Create() = (%+v, %v, %v), want collision retry", link, isNew, err)
	}
}

func TestURLShortenerResolvesAndCountsHit(t *testing.T) {
	store := newMemoryRepository()
	shortener := newTestService(t, store, "AbCd1234")
	link, _, err := shortener.Create(context.Background(), "https://example.com/target")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := shortener.Resolve(context.Background(), link.Shortened)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Original != link.Original || resolved.Hits != 1 {
		t.Fatalf("Resolve() = %+v, want target and one hit", resolved)
	}
}

func TestURLShortenerRejectsInvalidOrExpiredLinks(t *testing.T) {
	store := newMemoryRepository()
	store.bySlug["Expired1"] = models.ShortLink{
		Shortened: "Expired1",
		Original:  "https://example.com",
		Created:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Expiry:    30,
	}
	shortener := newTestService(t, store, "Unused12")

	if _, _, err := shortener.Create(context.Background(), "javascript://example.com"); !errors.Is(err, service.ErrInvalidURL) {
		t.Fatalf("Create() error = %v, want ErrInvalidURL", err)
	}
	for _, slug := range []string{"bad", "Expired1", "../../etc/passwd"} {
		if _, err := shortener.Resolve(context.Background(), slug); !errors.Is(err, service.ErrNotFound) {
			t.Errorf("Resolve(%q) error = %v, want ErrNotFound", slug, err)
		}
	}
}

func TestURLShortenerRejectsInvalidGeneratorOutput(t *testing.T) {
	shortener := newTestService(t, newMemoryRepository(), "not-valid")
	if _, _, err := shortener.Create(context.Background(), "https://example.com"); err == nil {
		t.Fatal("Create() accepted invalid slug generator output")
	}
}
