// Package web contains a small web-framework extension.
package web

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/dimfeld/httptreemux/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// KeyValues is how request values are stored/retrieved.
const KeyValues ctxKey = 1

// Values represent state for each request.
type Values struct {
	TraceID    string
	Now        time.Time
	StatusCode int
}

// A Handler is a type that handles a http requests within our own tiny framework
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// An App is the entrypoint into our application and what configurates our context
// object for each of our http hanlers. Feel free to add any configuration
// data/logic on this App struct.
type App struct {
	mux      *httptreemux.ContextMux
	otmux    http.Handler
	shutdown chan os.Signal
	mw       []Middleware
}

// NewApp creates an App value that handle a set of routes for the application.
func NewApp(shutdown chan os.Signal, mw ...Middleware) *App {
	// Create and OpenTelemetry HTTP Handler which wraps the router. This will
	// start the initial span and annotate it with information about the
	// request/response.
	//
	// This is configured to use the W3C TraceContext standard to set the remote
	// parent if an client request includes the appropriate headers.
	// https://w3c.github.io/trace-context/

	mux := httptreemux.NewContextMux()
	return &App{
		mux:      mux,
		otmux:    otelhttp.NewHandler(mux, "request"),
		shutdown: shutdown,
		mw:       mw,
	}

}

// SignalShutdown is used to gracefully shutdown the app when an integrity
// issue is identified.
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}

// ServeHTTP implements the http.Handler interface. It's the entry point for
// all http traffic and allow the opentelemetry mux to run first to handle
// tracing. The opentelemetry mux then calls the application mux to handle
// appkication traffic. This was setup in the NewApp function.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.otmux.ServeHTTP(w, r)
}

func (a *App) Handle(method string, path string, handler Handler, mw ...Middleware) {

	// First wrap handler specific middleware around this handler
	handler = wrapMiddleware(mw, handler)

	// Add the application's general middleware to the handler chain.
	handler = wrapMiddleware(a.mw, handler)

	h := func(w http.ResponseWriter, r *http.Request) {

		// Start the context with the required values to process the request.
		ctx := r.Context()
		ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("web").Start(ctx, r.URL.Path)
		defer span.End()
		// Set the context  with the required values to process the request.
		v := Values{
			TraceID: span.SpanContext().TraceID().String(),
			Now:     time.Now(),
		}

		ctx = context.WithValue(ctx, KeyValues, &v)

		err := handler(ctx, w, r)
		if err != nil {
			a.SignalShutdown()
			return
		}
	}

	a.mux.Handle(method, path, h)
}
