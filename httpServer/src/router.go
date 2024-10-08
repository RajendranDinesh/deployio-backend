package src

import (
	auth "httpServer/src/routes/Auth"
	build "httpServer/src/routes/Build"
	deployment "httpServer/src/routes/Deployment"
	github "httpServer/src/routes/Github"
	project "httpServer/src/routes/Project"
	user "httpServer/src/routes/User"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Service() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.RealIP)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	router.Mount("/api/v1/auth", auth.AuthRouter())
	router.Mount("/api/v1/dashboard", user.UserRouter())
	router.Mount("/api/v1/github", github.GithubRouter())
	router.Mount("/api/v1/project", project.ProjectRouter())
	router.Mount("/api/v1/build", build.BuildRouter())
	router.Mount("/api/v1/deployment", deployment.DeploymentRouter())

	router.Handle("/metrics", promhttp.Handler())

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})

	return router
}
