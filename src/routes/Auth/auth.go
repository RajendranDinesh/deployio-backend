package auth

import "github.com/go-chi/chi/v5"

func AuthRouter() chi.Router {
	r := chi.NewRouter()

	u := AuthHandler{}

	r.Post("/signin", u.SignIn)

	return r
}
