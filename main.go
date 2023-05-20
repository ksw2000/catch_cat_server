package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/ksw2000/catch_cat_server/config"
	"github.com/ksw2000/catch_cat_server/session"
	"github.com/ksw2000/catch_cat_server/util"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	r := gin.Default()
	r.Use(CORSMiddleware())

	r.POST("/register", postRegister)
	r.POST("/login", postLogin)
	r.POST("/logout", postLogout)
	r.GET("/me", getMe)
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

func postRegister(c *gin.Context) {
	req := struct {
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
		Email           string `json:"email"`
		Name            string `json:"name"`
	}{}
	res := struct {
		Error string `json:"error"`
	}{}

	if err := c.BindJSON(&req); err != nil {
		log.Print(err)
		return
	}
	if len(req.Name) > 12 && len(req.Name) <= 0 {
		res.Error = "名稱需介於 1~12 字元"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if req.ConfirmPassword != req.Password {
		res.Error = "密碼與確認密碼不符"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if err := util.PasswordFormatChecker(req.Password); err != nil {
		res.Error = err.Error()
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// TODO: check email
	// TODO: send email

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()

	// check if there are the same email in db
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM user WHERE `email` = ?", req.Email)
	if err := row.Scan(&count); err != nil {
		res.Error = fmt.Sprintf("db.QueryRow() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	} else if count > 0 {
		res.Error = "Email 已經註冊"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// check if there are the same user_id in db
	uid := uint64(rand.Float64()*900000000000. + 100000000000.)

	for count := 1; count != 0; {
		row := db.QueryRow("SELECT COUNT(*) FROM user WHERE `user_id` = ?", uid)
		if err := row.Scan(&count); err != nil {
			res.Error = fmt.Sprintf("db.QueryRow() error %v", err)
			c.IndentedJSON(http.StatusOK, res)
			return
		}

		// generate again
		uid = uint64(rand.Float64()*900000000000. + 100000000000.)
	}

	stmt, err := db.Prepare("INSERT INTO user(user_id, salt, password, name, profile, email, creating, last_login, verified) values(?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	salt := util.RandomString(256)
	hashedPassword := util.PasswordHash(req.Password, salt)
	if _, err = stmt.Exec(uid, salt, hashedPassword, req.Name, "", req.Email, time.Now().Unix(), time.Now().Unix(), false); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	c.IndentedJSON(http.StatusCreated, res)
}

func postLogin(c *gin.Context) {
	req := struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}{}

	res := struct {
		Error    string `json:"error"`
		Session  string `json:"session"`
		Name     string `json:"name"`
		Uid      uint64 `json:"uid"`
		Profile  string `json:"profile"`
		Email    string `json:"email"`
		Rank     int    `json:"rank"`
		Verified bool   `json:"verified"`
		Cats     int    `json:"cats"`
	}{}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()

	var hashedPassword, salt string
	row := db.QueryRow("SELECT password, salt FROM user WHERE `email` = ?", req.Email)
	if err := row.Scan(&hashedPassword, &salt); err != nil {
		res.Error = fmt.Sprintf("尚未註冊 %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if hashedPassword != util.PasswordHash(req.Password, salt) {
		res.Error = "密碼錯誤"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	row = db.QueryRow("SELECT name, user_id, profile, email, verified FROM user WHERE `email` = ?", req.Email)
	if err := row.Scan(&res.Name, &res.Uid, &res.Profile, &res.Email, &res.Verified); err != nil {
		res.Error = fmt.Sprintf("database error row.Scan() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// TODO
	res.Rank = 0
	res.Cats = 0
	res.Session = session.NewSession()
	val, _ := session.Get(res.Session)
	val["uid"] = res.Uid

	c.IndentedJSON(http.StatusOK, res)
}

func postLogout(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
	}{}
	res := struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}{}
	if err := c.BindJSON(&req); err != nil {
		return
	}

	val, ok := session.Get(req.Session)
	if !ok {
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	uid := val["uid"].(uint64)

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()

	stmt, _ := db.Prepare("UPDATE user SET last_login=? WHERE user_id=?")
	if _, err := stmt.Exec(time.Now().Unix(), uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	res.Ok = true
	c.IndentedJSON(http.StatusOK, res)
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

func getMe(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
	}{}
	res := struct {
		IsLogin  bool   `json:"is_login"`
		Error    string `json:"error"`
		Name     string `json:"name"`
		Uid      uint64 `json:"uid"`
		Profile  string `json:"profile"`
		Email    string `json:"email"`
		Rank     int    `json:"rank"`
		Verified bool   `json:"verified"`
		Cats     int    `json:"cats"`
	}{}
	if err := c.BindJSON(&req); err != nil {
		return
	}

	val, ok := session.Get(req.Session)
	if !ok {
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()

	res.Uid = val["uid"].(uint64)
	row := db.QueryRow("SELECT name, profile, email, verified FROM user WHERE `user_id` = ?", res.Uid)
	if err := row.Scan(&res.Name, &res.Profile, &res.Email, &res.Email, &res.Verified); err != nil {
		res.Error = "database error row.Scan() error"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// TODO
	res.Rank = 0
	res.Cats = 0

	res.IsLogin = true
	c.IndentedJSON(http.StatusOK, res)
}
