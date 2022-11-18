package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/pavel418890/service/business/auth"
	"github.com/pavel418890/service/business/mid"

	"github.com/pavel418890/service/foundation/web"
)

func API(build string, shutdown chan os.Signal, log *log.Logger, a *auth.Auth) *web.App {

	app := web.NewApp(shutdown, mid.Logger(log), mid.Errors(log), mid.Metrics(), mid.Panics(log))

	check := check{
		log: log,
	}

	app.Handle(http.MethodGet, "/readiness", check.readiness, mid.Authenticate(a), mid.Authorize(log, auth.RoleAdmin))
	app.Handle(http.MethodGet, "/liveness", check.readiness, mid.Authenticate(a), mid.Authorize(log, auth.RoleAdmin))

	return app

}
