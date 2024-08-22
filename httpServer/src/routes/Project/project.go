package project

import (
	"httpServer/src/middleware"
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

		r.Use(middleware.GithubTokenValidation)

		r.Get("/all", p.ListProjects)
		r.Post("/new", p.CreateNewProject)
		r.Get("/{id}", p.Project)
		r.Get("/environments/{id}", p.ListEnvKeys)
		r.Post("/environments", p.InsertEnvironments)
		r.Put("/environments", p.UpdateEnvValue)
		r.Delete("/environments", p.DeleteEnv)
		r.Delete("/{projectId}", p.DeleteProject)
	})

	return r
}
