package deployment

import (
	auth "httpServer/src/routes/Auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func DeploymentRouter() chi.Router {
	r := chi.NewRouter()

	d := DeploymentHandler{}

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(auth.GetJWTAuthConfig()))
		r.Use(jwtauth.Authenticator(auth.GetJWTAuthConfig()))

		r.Get("/all", d.ListDeployments)
		r.Delete("/", d.DeleteDeployment)
	})

	return r
}
