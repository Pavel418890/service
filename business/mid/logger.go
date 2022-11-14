package mid

import (
	"context"
	"net/http"

	"github.com/pavel4188890/service/foundation/web"
)

// Logger ...
func Logger(log *log.Logger) web.Middleware {
	m := func(before web.Handler) web.Handler {

		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			err := before(ctx, w, r)

			log.Println(r, status)

			return err
		}

		return h
	}

	return m
}
