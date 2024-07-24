package middleware

import (
	"deployio-backend/config"
	auth "deployio-backend/src/routes/Auth"
	"deployio-backend/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/jwtauth/v5"
)

func GithubTokenValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, claims, _ := jwtauth.FromContext(r.Context())

		userId, err := strconv.Atoi(fmt.Sprintf("%v", claims["uId"]))

		if err != nil {
			println("[TOKENS] Error Converting string to integer")
			println(err.Error())
			return
		}

		isAccessValid, isRefreshValid := auth.AreTokensValid(userId)

		if isAccessValid && isRefreshValid {
			next.ServeHTTP(w, r)
			return
		} else if isRefreshValid {
			var TokenPayload auth.UserSignInPayload
			var refreshToken string

			query := `SELECT refresh FROM "deploy-io".users WHERE id = $1`
			err := config.DataBase.QueryRow(query, userId).Scan(&refreshToken)
			if err != nil {
				println(err.Error())
				println("[USER] Error while fetching refresh token")
				utils.ForcedRelogin(w)
				return
			}

			TokenPayload.RefreshToken = refreshToken
			cId, cSecret := auth.GetClientIdnSecret()

			response, err := auth.GetOauthResponse(cId, cSecret, TokenPayload)
			if err != nil {
				println(err.Error())
				println("[USER] Refresh id invalid")
				utils.ForcedRelogin(w)
				return
			}

			auth.UpdateUserTokens(response, int64(userId))
			next.ServeHTTP(w, r)
			return
		} else {
			utils.ForcedRelogin(w)
			return
		}
	})
}
