package github

import (
	"httpServer/src/middleware"
	auth "httpServer/src/routes/Auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func GithubRouter() chi.Router {
	r := chi.NewRouter()

	gHandler := GithubHandler{}

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(auth.GetJWTAuthConfig()))
		r.Use(jwtauth.Authenticator(auth.GetJWTAuthConfig()))

		r.Use(middleware.GithubTokenValidation)

		r.Get("/repos", gHandler.GetUserRepositories)
	})

	return r
}
