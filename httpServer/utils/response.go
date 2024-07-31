package utils

import (
	"net/http"
)

type ErrorType struct {
	Code        int
	Description string
}

var (
	ErrUnAuthorized = ErrorType{http.StatusUnauthorized, "Do I know you?"}
	ErrInvalid      = ErrorType{http.StatusBadRequest, "Invalid request"}
	TokenExpired    = ErrorType{498, "Trying to imitate someone?"}
	ErrInternal     = ErrorType{http.StatusInternalServerError, "Internal Server Error"}
)

func HandleError(errType ErrorType, err error, w http.ResponseWriter) {
	if err != nil {
		println(err.Error())
		http.Error(w, errType.Description, errType.Code)
	}
}
