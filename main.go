package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/peterbourgon/g2s"
)

const (
	bufferSize  = 10
	bits        = 2048
	timeout     = 500 * time.Millisecond
	concurrency = 5
)

var (
	statsd, err = g2s.Dial("udp", "statsd-server:8125")
)

func GeneratePrimes(ch chan<- *big.Int) {
	for {
		p, err := rand.Prime(rand.Reader, bits)
		if err != nil {
			panic(err)
		}
		ch <- p
	}
}

func MakeHandler(ch <-chan *big.Int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		defer func(begun time.Time) {
			statsd.Timing(1, "prime.duration", time.Since(begun))
			statsd.Counter(1, fmt.Sprintf("prime.counter.%d", status), 1)
		}(time.Now())
		select {
		case p := <-ch:
			fmt.Fprintln(w, p.Text(16))
		case <-time.After(timeout):
			status = http.StatusServiceUnavailable
			http.Error(
				w,
				fmt.Sprint("timeout reached after ", timeout),
				status,
			)
		}
	}
}

func main() {
	ch := make(chan *big.Int, bufferSize)
	for i := 0; i < concurrency; i++ {
		go GeneratePrimes(ch)
	}
	http.HandleFunc("/prime", MakeHandler(ch))
	http.ListenAndServe(":8080", nil)
}
