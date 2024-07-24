package utils

import "net/http"

func ErrInternalServer(err error, w http.ResponseWriter) {
	println("[AUTH] ", err.Error())
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func ErrInvalid(err error, w http.ResponseWriter) {
	http.Error(w, "Invalid Request", http.StatusBadRequest)
}

func ForcedRelogin(w http.ResponseWriter) {
	http.Error(w, "Token expired", 498)
}
