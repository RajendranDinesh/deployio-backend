package user

import (
	"deployio-backend/src/middleware"
	auth "deployio-backend/src/routes/Auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func UserRouter() chi.Router {
	r := chi.NewRouter()

	u := UserHandler{}

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(auth.GetJWTAuthConfig()))
		r.Use(jwtauth.Authenticator(auth.GetJWTAuthConfig()))

		r.Use(middleware.GithubTokenValidation)
		r.Get("/", u.GetDashboardDetails)
	})

	return r
}
