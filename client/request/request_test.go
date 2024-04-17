package request

import (
	"sync"
	"testing"
)

func BenchmarkRequest(b *testing.B) {
	wg := &sync.WaitGroup{}
	client := GetClient()

	wg.Add(b.N)

	for i := 0; i < b.N; i++ {
		go Request(wg, client)
	}

	wg.Wait()
}
