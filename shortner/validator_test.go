package shortner

import (
	"strings"
	"testing"
)

func TestURLValidatorAcceptsHTTPURLs(t *testing.T) {
	t.Parallel()
	validator := NewURLValidator()
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
	validator := NewURLValidator()
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
		"https://example.com/" + strings.Repeat("a", MaxURLLength),
	} {
		if err := validator.Validate(candidate); err == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", candidate)
		}
	}
}

func TestShortenUsesExpectedSlugFormat(t *testing.T) {
	t.Parallel()
	slug, err := Shorten("https://example.com/a/long/path")
	if err != nil {
		t.Fatal(err)
	}
	if !validSlug.MatchString(slug) {
		t.Fatalf("generated slug %q does not match the public format", slug)
	}
}

func TestCryptoSlugGeneratorProducesDistinctValues(t *testing.T) {
	t.Parallel()
	generator := CryptoSlugGenerator{Length: DefaultSlugLength}
	seen := make(map[string]struct{}, 1_000)
	for index := 0; index < 1_000; index++ {
		slug, err := generator.Generate()
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := seen[slug]; exists {
			t.Fatalf("unexpected duplicate slug %q", slug)
		}
		seen[slug] = struct{}{}
	}
}
