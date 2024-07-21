package src

import (
	auth "deployio-backend/src/routes/Auth"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func Service() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.RealIP)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.Mount("/api/v1/auth", auth.AuthRouter())

	return router
}
