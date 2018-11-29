package middleware

import (
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fasthttp"
	"strings"
)

type CustomClaims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

type Auth struct {
}

const (
	tokenSignature = "henghajiangiscoming20161010"
)

var (
	authLogger = log.New()
)

func (a *Auth) Work(ctx *fasthttp.RequestCtx, errChan chan error) {
	var token string
	var authorization string

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		if string(key) == "Authorization" {
			authorization = string(value)
		}
	})
	if authorization != "" {
		tokenList := strings.Split(authorization, " ")
		if len(tokenList) == 2 {
			token = tokenList[1]
		}
	}

	ctx.SetUserValue("is_logged_in", false)
	ctx.SetUserValue("user_id", 0)

	if token != "" {
		_, err := jwt.ParseWithClaims(token, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(tokenSignature), nil
		})
		if err != nil {
			authLogger.Exception(err)
		} else {
			//hhjConn.Where(&hhj.Tokens{Token: token}).First(&tokens)
			//if claims, ok := t.Claims.(*CustomClaims); ok && t.Valid {
			//	if claims.UserID == tokens.UserID {
			//		// authorization passed
			//		log.Print(claims.UserID)
			//		ctx.SetUserValue("is_logged_in", true)
			//		ctx.SetUserValue("user_id", claims.UserID)
			//	}
			//}
		}
	}
}
