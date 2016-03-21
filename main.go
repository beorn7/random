package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

const (
	bufferSize = 10
	bits       = 2048
	timeout    = 500 * time.Millisecond
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
		select {
		case p := <-ch:
			fmt.Fprintln(w, p.Text(16))
		case <-time.After(timeout):
			http.Error(
				w,
				fmt.Sprint("timeout reached after ", timeout),
				http.StatusServiceUnavailable,
			)
		}
	}
}

func main() {
	ch := make(chan *big.Int, bufferSize)
	go GeneratePrimes(ch)
	http.HandleFunc("/prime", MakeHandler(ch))
	http.ListenAndServe(":8080", nil)
}
