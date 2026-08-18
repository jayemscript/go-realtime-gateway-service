package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, requestID)))
	})
}

func newRequestID() string {
	var randomBytes [12]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "req_unknown"
	}
	return "req_" + hex.EncodeToString(randomBytes[:])
}
