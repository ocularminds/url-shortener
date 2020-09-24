package shortner

import (
	"strconv"
	"testing"
)

var result string

func BenchmarkFibComplete(b *testing.B) {
	var r string
	for n := 0; n < b.N; n++ {
		// always record the result of Fib to prevent
		// the compiler eliminating the function call.
		r, err := Shorten("http://stord.com/" + strconv.Itoa(n))
		if err != nil {
			result = r
		}
	}
	// always store the result to a package level variable
	// so the compiler cannot eliminate the Benchmark itself.
	result = r
}
