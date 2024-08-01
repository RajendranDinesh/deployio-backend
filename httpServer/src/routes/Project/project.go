package project

import (
	auth "httpServer/src/routes/Auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func ProjectRouter() chi.Router {
	r := chi.NewRouter()

	p := ProjectHandler{}

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(auth.GetJWTAuthConfig()))
		r.Use(jwtauth.Authenticator(auth.GetJWTAuthConfig()))

		r.Get("/all", p.ListProjects)
		r.Post("/new", p.CreateNewProject)
	})

	return r
}
