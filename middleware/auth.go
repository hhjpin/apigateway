package middleware

import (
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fasthttp"
	"strings"
	"github.com/gomodule/redigo/redis"

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
	tokenSignature = "henghajiangiscoming20161010"
	token2user    = "hhj:token:user"

)

var (
	authLogger = log.New()
)
var redisConn redis.Conn


func init() {
	var auth Auth

	redisConn, err := redis.Dial("tcp", auth.Redis.Addr, redis.DialPassword(auth.Redis.Password))
	if err != nil {
		authLogger.Exception(err)
	}
	_, err = redisConn.Do("SELECT", auth.Redis.Database)
	if err != nil {
		authLogger.Exception(err)
	}
}

func GetUserByToken(token string) (userId int, err error) {
	userId, err = redis.Int(redisConn.Do("HGET", token2user, token))
	return
}


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
		t, err := jwt.ParseWithClaims(token, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(tokenSignature), nil
		})
		if err != nil {
			authLogger.Exception(err)
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
