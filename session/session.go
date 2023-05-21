package session

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ksw2000/catch_cat_server/util"
)

type Session struct {
	value map[string]interface{}
}

var bucket map[string]*Session

func NewSession() (token string) {
	if bucket == nil {
		bucket = make(map[string]*Session)
	}
	token = util.RandomString(256)

	// if token has been existed
	for _, ok := bucket[token]; ok; {
		token = util.RandomString(256)
		_, ok = bucket[token]
	}

	bucket[token] = &Session{
		value: map[string]interface{}{},
	}
	return token
}

func Get(token string) (value map[string]interface{}, ok bool) {
	session, ok := bucket[token]
	if !ok {
		return nil, ok
	}
	return session.value, ok
}

func Destroy(token string) {
	delete(bucket, token)
}

func CheckLogin(c *gin.Context, sessionID string) (uid uint64, isLogin bool) {
	val, isLogin := Get(sessionID)
	if !isLogin {
		c.IndentedJSON(http.StatusUnauthorized, struct {
			Error string `json:"error"`
		}{"未登入"})
		return
	}

	uid = val["uid"].(uint64)
	return
}
