package shortner

import "testing"

var benchmarkSlug string

func BenchmarkCryptoSlugGenerator(b *testing.B) {
	generator := CryptoSlugGenerator{Length: DefaultSlugLength}
	for index := 0; index < b.N; index++ {
		slug, err := generator.Generate()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSlug = slug
	}
}
