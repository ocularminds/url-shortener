package tests

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ocularminds/url-shortener/core/service"
)

func TestURLValidatorAcceptsHTTPURLs(t *testing.T) {
	t.Parallel()
	validator := service.NewURLValidator()
	for _, candidate := range []string{
		"https://www.example.com/search?q=url+shortener#results",
		"http://localhost:8080/path",
		"https://xn--bcher-kva.example/path",
	} {
		if err := validator.Validate(candidate); err != nil {
			t.Errorf("Validate(%q) returned %v", candidate, err)
		}
	}
}

func TestURLValidatorRejectsUnsafeOrMalformedURLs(t *testing.T) {
	t.Parallel()
	validator := service.NewURLValidator()
	for _, candidate := range []string{
		"",
		"example.com/path",
		"javascript://example.com/alert(1)",
		"ftp://example.com/file",
		"https://user:password@example.com",
		" https://example.com",
		"https://example.com\r\nX-Injected: true",
		"https://",
		"https://example.com:" + strings.Repeat("9", 8),
		"https://example.com/" + strings.Repeat("a", service.MaxURLLength),
	} {
		if err := validator.Validate(candidate); err == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", candidate)
		}
	}
}

func TestCryptoSlugGeneratorUsesExpectedFormat(t *testing.T) {
	t.Parallel()
	generator := service.CryptoSlugGenerator{Length: service.DefaultSlugLength}
	format := regexp.MustCompile(`^[A-Za-z0-9]{8}$`)
	seen := make(map[string]struct{}, 1_000)
	for index := 0; index < 1_000; index++ {
		slug, err := generator.Generate()
		if err != nil {
			t.Fatal(err)
		}
		if !format.MatchString(slug) {
			t.Fatalf("generated slug %q does not match public format", slug)
		}
		if _, exists := seen[slug]; exists {
			t.Fatalf("unexpected duplicate slug %q", slug)
		}
		seen[slug] = struct{}{}
	}
}

func TestURLValidatorUsesDefaultLimitAndRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()
	validator := service.URLValidator{}
	if err := validator.Validate("https://example.com"); err != nil {
		t.Fatalf("zero-value validator rejected a valid URL: %v", err)
	}
	if err := validator.Validate(string([]byte("https://example.com/\xff"))); err == nil {
		t.Fatal("validator accepted invalid UTF-8")
	}
	if !service.IsValid("https://example.com") || service.IsValid("mailto:user@example.com") {
		t.Fatal("IsValid() returned an unexpected result")
	}
}

func TestCryptoSlugGeneratorUsesDefaultLength(t *testing.T) {
	t.Parallel()
	slug, err := (service.CryptoSlugGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(slug) != service.DefaultSlugLength {
		t.Fatalf("generated slug length = %d", len(slug))
	}
}
