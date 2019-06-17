package middleware

import (
	"git.henghajiang.com/backend/api_gateway_v2/middleware/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/gomodule/redigo/redis"
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

type CustomClaims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

type Auth struct {
	Redis struct {
		Addr     string `json:"addr"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"redis"`
}

const (
	tokenSignature = "your token signature"
	token2user     = "token:user"
)

var redisConn redis.Conn

func GetUserByToken(token string) (userId int, err error) {
	userId, err = redis.Int(redisConn.Do("HGET", token2user, token))
	return
}

func (a *Auth) Work(ctx *fasthttp.RequestCtx, errChan chan error) {
	defer func() {
		if err := recover(); err != nil {
			stack := utils.Stack(3)
			logger.Errorf("[Recovery] %s panic recovered:\n%s\n%s", utils.TimeFormat(time.Now()), err, stack)
		}
	}()

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
		t, err := jwt.ParseWithClaims(token, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(tokenSignature), nil
		})
		if err != nil {
			logger.Exception(err)
		} else {

			userId, _ := GetUserByToken(token)
			if claims, ok := t.Claims.(*CustomClaims); ok && t.Valid {
				if claims.UserID == userId {
					// authorization passed
					ctx.SetUserValue("is_logged_in", true)
					ctx.SetUserValue("user_id", claims.UserID)
				}
			}
		}
	}
}
