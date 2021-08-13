// Copyright 2021 Hewlett Packard Enterprise Development LP

package tokens

import (
	"log"
	"net/http"
	"time"
)

// Logger is the general api logger
func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}
