package build

import (
	"httpServer/src/middleware"
	auth "httpServer/src/routes/Auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func BuildRouter() chi.Router {
	r := chi.NewRouter()

	u := BuildHandler{}

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(auth.GetJWTAuthConfig()))
		r.Use(jwtauth.Authenticator(auth.GetJWTAuthConfig()))

		r.Use(middleware.GithubTokenValidation)

		r.Post("/new", u.CreateBuild)
		r.Get("/all", u.ListBuilds)
		r.Get("/", u.Build)
	})

	return r
}
