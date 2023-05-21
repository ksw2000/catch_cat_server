package user

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

func GetMe(c *gin.Context) {
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
		Verified bool   `json:"verified"`
		Score    int    `json:"score"`
		Level    int    `json:"level"`
		Cats     int    `json:"cats"`
	}{}
	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		return
	}
	res.Uid = uid

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()

	row := db.QueryRow("SELECT name, profile, email, verified FROM user WHERE `user_id` = ?", res.Uid)
	if err := row.Scan(&res.Name, &res.Profile, &res.Email, &res.Email, &res.Verified); err != nil {
		res.Error = "database error row.Scan() error"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	res.Cats, res.Score, res.Level = getScoreAndLevel(db, res.Uid)

	res.IsLogin = true
	c.IndentedJSON(http.StatusOK, res)
}

func PostRegister(c *gin.Context) {
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

func PostLogin(c *gin.Context) {
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
		Verified bool   `json:"verified"`
		Score    int    `json:"score"`
		Level    int    `json:"level"`
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

	res.Cats, res.Score, res.Level = getScoreAndLevel(db, res.Uid)

	res.Session = session.NewSession()
	val, _ := session.Get(res.Session)
	val["uid"] = res.Uid

	c.IndentedJSON(http.StatusOK, res)
}

func PostLogout(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
	}{}
	res := struct {
		Error string `json:"error"`
	}{}
	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		return
	}

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

	c.IndentedJSON(http.StatusOK, res)
}

func getScoreAndLevel(db *sql.DB, uid uint64) (cats int, score int, level int) {
	// 1 level = 100 score
	row := db.QueryRow(`
		SELECT SUM(cat_kind.weight), COUNT(user_cat.cat_id)
		FROM cat_kind, cat, user_cat 
		WHERE 
		user_cat.cat_id = cat.cat_id and 
		cat.cat_kind_id = cat_kind.cat_kind_id and
		user_cat.user_id = ?`, uid)
	row.Scan(&score, &cats)
	level = score / 100
	return cats, score, level
}
