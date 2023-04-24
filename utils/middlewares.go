package utils

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/apex/log"
)

func MiddlewareWrap(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

type ContextKey struct {
	name string
}

func (key ContextKey) String() string {
	return key.name
}

var RequestCtxKey = &ContextKey{name: "RequestContext"}

type RequestContext struct {
	RequestLog *HttpRequestLog
	LogEntry   *log.Entry
}

type HttpRequestLog struct {
	Timestamp     time.Time `json:"-"`
	Latency       string    `json:"latency,omitempty"`
	Protocol      string    `json:"protocol,omitempty"`
	Referer       string    `json:"referer,omitempty"`
	RemoteIp      string    `json:"remoteIp,omitempty"`
	RequestMethod string    `json:"requestMethod,omitempty"`
	RequestUrl    string    `json:"requestUrl,omitempty"`
	UserAgent     string    `json:"userAgent,omitempty"`
	ServerIp      string    `json:"serverIp,omitempty"`
}

func (reqLog *HttpRequestLog) Finalize() {
	reqLog.Latency = time.Since(reqLog.Timestamp).String()
}

func LogAndRecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		reqLog := &HttpRequestLog{
			Timestamp:     time.Now(),
			Protocol:      request.Proto,
			Referer:       request.Referer(),
			RemoteIp:      request.Header.Get("X-Forwarded-For"),
			RequestMethod: request.Method,
			RequestUrl:    request.RequestURI,
			UserAgent:     request.UserAgent(),
			ServerIp:      request.RemoteAddr,
		}
		reqCtx := &RequestContext{
			RequestLog: reqLog,
			LogEntry:   log.WithField("httpRequest", reqLog),
		}

		defer func() {
			if rvr := recover(); rvr != nil {
				if request.ParseForm() == nil {
					reqCtx.LogEntry = reqCtx.LogEntry.WithField("httpForm", request.Form)
				}
				reqLog.Finalize()
				reqCtx.LogEntry.WithField("panic", rvr).
					WithField("stackTrace", string(debug.Stack())).
					Error("Handler panicked")

				http.Error(writer, "internal server error", http.StatusInternalServerError)
			}
		}()

		ctx := context.WithValue(request.Context(), RequestCtxKey, reqCtx)
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
