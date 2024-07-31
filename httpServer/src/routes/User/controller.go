package user

import (
	"net/http"
)

func (u UserHandler) GetDashboardDetails(w http.ResponseWriter, r *http.Request) {
	// _, claims, _ := jwtauth.FromContext(r.Context())

	w.Write([]byte("came through"))
}
