package mid

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/pavel418890/service/foundation/web"
)

// Logger writes some info about the request to the logs in the format
// TraceID : (200) GET /route -> IP ADDR (latency)
func Logger(log *log.Logger) web.Middleware {

	// This is the actual middleware function to be executed
	m := func(handler web.Handler) web.Handler {

		// Create the handler that will be attached in the middleware chain.
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			// If the context is missing this value ,request the service
			// to be shutdown gracefully.
			v, ok := ctx.Value(web.KeyValues).(*web.Values)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			log.Printf("%s : started    : %s %s -> %s",
				v.TraceID, r.Method, r.URL.Path, r.RemoteAddr,
			)
			err := handler(ctx, w, r)

			log.Printf("%s : completed  : %s %s -> %s (%d) (%s)",
				v.TraceID,
				r.Method, r.URL.Path, r.RemoteAddr,
				v.StatusCode, time.Since(v.Now),
			)

			return err
		}

		return h
	}

	return m
}
