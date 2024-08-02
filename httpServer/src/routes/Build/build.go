package build

import "github.com/go-chi/chi/v5"

func BuildRouter() chi.Router {
	r := chi.NewRouter()

	u := BuildHandler{}

	r.Post("/new", u.CreateBuild)
	r.Get("/all", u.ListBuilds)
	r.Get("/:id", u.Build)

	return r
}
