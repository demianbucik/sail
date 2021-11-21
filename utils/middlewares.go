package utils

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/apex/log"
)

type ContextKey struct {
	name string
}

func (key ContextKey) String() string {
	return key.name
}

var LogEntryCtxKey = &ContextKey{name: "LogEntry"}

func ApplyMiddlewares(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

func LogEntryAndRecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		logEntry := log.WithField("userAgent", request.UserAgent()).
			WithField("remoteAddr", request.Header.Get("X-Forwarded-For")).
			WithField("url", request.RequestURI).
			WithField("host", request.Host).
			WithField("headers", request.Header)

		defer func() {
			if rvr := recover(); rvr != nil {
				logEntry.WithField("panic", rvr).
					WithField("stackTrace", string(debug.Stack())).
					Error("Handler panicked")

				http.Error(writer, "internal server error", http.StatusInternalServerError)
			}
		}()

		ctx := context.WithValue(request.Context(), LogEntryCtxKey, logEntry)
		next.ServeHTTP(writer, request.WithContext(ctx))
	}
	return fn
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		if origin := request.Header.Get("Origin"); origin != "" {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "*")
		if request.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(writer, request)
	}
	return fn
}
