package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/ksw2000/catch_cat_server/cats"
	"github.com/ksw2000/catch_cat_server/config"
	"github.com/ksw2000/catch_cat_server/friends"
	"github.com/ksw2000/catch_cat_server/user"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	r := gin.Default()
	r.Use(CORSMiddleware())

	r.POST("/register", user.PostRegister)
	r.POST("/login", user.PostLogin)
	r.POST("/logout", user.PostLogout)
	r.POST("/friend/invite", friends.PostFriendInvite)
	r.POST("/friends/inviting_me", friends.PostInvitingMeList)
	r.POST("/friends/list", friends.PostFriendsList)
	r.POST("/friend/agree", friends.PostFriendAgree)
	r.POST("/friend/decline", friends.PostFriendDecline)
	r.POST("/friend/delete", friends.PostFriendDelete)
	r.POST("/theme", cats.PostTheme)
	r.GET("/me", user.GetMe)
	r.GET("/theme_list", getThemeList)
	r.Run("localhost:8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// https://stackoverflow.com/questions/29418478/go-gin-framework-cors
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func getThemeList(c *gin.Context) {
	type Theme struct {
		ThemeID     int    `json:"theme_id"`
		Name        string `json:"name"`
		Thumbnail   string `json:"thumbnail"`
		Description string `json:"description"`
	}
	res := struct {
		Error string  `json:"error"`
		List  []Theme `json:"list"`
	}{
		List: []Theme{},
	}

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT theme_id, name, thumbnail, description FROM theme")
	if err != nil {
		res.Error = fmt.Sprintf("db.Query() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	for rows.Next() {
		theme := Theme{}
		rows.Scan(&theme.ThemeID, &theme.Name, &theme.Thumbnail, &theme.Description)
		res.List = append(res.List, theme)
	}

	c.IndentedJSON(http.StatusOK, res)
}
