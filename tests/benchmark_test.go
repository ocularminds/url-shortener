package tests

import (
	"testing"

	"github.com/ocularminds/url-shortener/core/service"
)

var benchmarkSlug string

func BenchmarkCryptoSlugGenerator(b *testing.B) {
	generator := service.CryptoSlugGenerator{Length: service.DefaultSlugLength}
	for index := 0; index < b.N; index++ {
		slug, err := generator.Generate()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSlug = slug
	}
}
