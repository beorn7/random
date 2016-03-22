package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	bufferSize  = 10
	bits        = 2048
	timeout     = 500 * time.Millisecond
	concurrency = 5
)

var (
	duration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "prime_request_duration_seconds",
		Help: "Histogram of the prime request duration.",
	})
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prime_requests_total",
			Help: "Total number of prime requests.",
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(duration)
	prometheus.MustRegister(counter)
}

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
			duration.Observe(time.Since(begun).Seconds())
			counter.With(prometheus.Labels{
				"status": fmt.Sprint(status),
			}).Inc()
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
	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(":8080", nil)
}
