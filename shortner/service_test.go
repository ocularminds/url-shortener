package shortner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type memoryRepository struct {
	mu      sync.Mutex
	bySlug  map[string]ShortLink
	created int
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{bySlug: make(map[string]ShortLink)}
}

func (repository *memoryRepository) FindBySlug(_ context.Context, slug string) (ShortLink, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	link, found := repository.bySlug[slug]
	if !found {
		return ShortLink{}, ErrNotFound
	}
	return link, nil
}

func (repository *memoryRepository) FindByOriginal(_ context.Context, original string) (ShortLink, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	for _, link := range repository.bySlug {
		if link.Original == original {
			return link, nil
		}
	}
	return ShortLink{}, ErrNotFound
}

func (repository *memoryRepository) Create(_ context.Context, link ShortLink) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, found := repository.bySlug[link.Shortened]; found {
		return ErrConflict
	}
	repository.bySlug[link.Shortened] = link
	repository.created++
	return nil
}

func (repository *memoryRepository) IncrementHits(_ context.Context, slug string) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	link, found := repository.bySlug[slug]
	if !found {
		return ErrNotFound
	}
	link.Hits++
	repository.bySlug[slug] = link
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

func newTestService(t *testing.T, repository LinkRepository, slugs ...string) *URLShortener {
	t.Helper()
	service, err := NewURLShortener(repository, &sequenceGenerator{values: slugs})
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC) }
	return service
}

func TestURLShortenerCreatesAndReusesLink(t *testing.T) {
	repository := newMemoryRepository()
	service := newTestService(t, repository, "AbCd1234")

	created, isNew, err := service.Create(context.Background(), "https://example.com/long")
	if err != nil || !isNew {
		t.Fatalf("first Create() = (%+v, %v, %v), want a new link", created, isNew, err)
	}
	existing, isNew, err := service.Create(context.Background(), "https://example.com/long")
	if err != nil || isNew || existing.Shortened != created.Shortened {
		t.Fatalf("second Create() = (%+v, %v, %v), want existing link", existing, isNew, err)
	}
	if repository.created != 1 {
		t.Fatalf("repository created %d records, want 1", repository.created)
	}
}

func TestURLShortenerRetriesSlugCollision(t *testing.T) {
	repository := newMemoryRepository()
	repository.bySlug["Collide1"] = ShortLink{Shortened: "Collide1", Original: "https://other.example"}
	service := newTestService(t, repository, "Collide1", "Unique12")

	link, isNew, err := service.Create(context.Background(), "https://example.com/long")
	if err != nil || !isNew || link.Shortened != "Unique12" {
		t.Fatalf("Create() = (%+v, %v, %v), want collision retry", link, isNew, err)
	}
}

func TestURLShortenerResolvesAndCountsHit(t *testing.T) {
	repository := newMemoryRepository()
	service := newTestService(t, repository, "AbCd1234")
	link, _, err := service.Create(context.Background(), "https://example.com/target")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := service.Resolve(context.Background(), link.Shortened)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Original != link.Original || resolved.Hits != 1 {
		t.Fatalf("Resolve() = %+v, want target and one hit", resolved)
	}
}

func TestURLShortenerRejectsInvalidOrExpiredLinks(t *testing.T) {
	repository := newMemoryRepository()
	repository.bySlug["Expired1"] = ShortLink{
		Shortened: "Expired1",
		Original:  "https://example.com",
		Created:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Expiry:    30,
	}
	service := newTestService(t, repository, "Unused12")

	if _, _, err := service.Create(context.Background(), "javascript://example.com"); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("Create() error = %v, want ErrInvalidURL", err)
	}
	for _, slug := range []string{"bad", "Expired1", "../../etc/passwd"} {
		if _, err := service.Resolve(context.Background(), slug); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%q) error = %v, want ErrNotFound", slug, err)
		}
	}
}
