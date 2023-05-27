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

func PostMe(c *gin.Context) {
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
		ShareGPS bool   `json:"share_gps"`
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

	row := db.QueryRow("SELECT name, profile, email, verified, share_gps FROM user WHERE user_id = ?", res.Uid)
	if err := row.Scan(&res.Name, &res.Profile, &res.Email, &res.Verified, &res.ShareGPS); err != nil {
		res.Error = "database error row.Scan() error"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	res.Cats, res.Score, res.Level = util.GetScoreAndLevel(db, res.Uid)

	res.IsLogin = true
	c.IndentedJSON(http.StatusOK, res)
}

func PostUpdateLastLogin(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
	}{}
	if err := c.BindJSON(&req); err != nil {
		return
	}
	res := struct {
		Error string `json:"error"`
	}{}

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

	stmt, err := db.Prepare("UPDATE user SET last_login = ? WHERE user_id = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(time.Now().Unix(), uid)
	if err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	c.IndentedJSON(http.StatusCreated, res)
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

	if err := checkName(req.Name); err != nil {
		res.Error = err.Error()
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if req.ConfirmPassword != req.Password {
		res.Error = "密碼與確認密碼不符"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if err := checkPasswordFormat(req.Password); err != nil {
		res.Error = err.Error()
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if err := checkEmailFormat(req.Email); err != nil {
		res.Error = err.Error()
		c.IndentedJSON(http.StatusOK, res)
		return
	}

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
		ShareGPS bool   `json:"share_gps"`
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

	row = db.QueryRow("SELECT name, user_id, profile, email, verified, share_gps FROM user WHERE `email` = ?", req.Email)
	if err := row.Scan(&res.Name, &res.Uid, &res.Profile, &res.Email, &res.Verified, &res.ShareGPS); err != nil {
		res.Error = fmt.Sprintf("database error row.Scan() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	res.Cats, res.Score, res.Level = util.GetScoreAndLevel(db, res.Uid)

	res.Session = session.NewSession()
	val, _ := session.Get(res.Session)
	val["uid"] = res.Uid

	c.IndentedJSON(http.StatusOK, res)
}

func PostUpdateName(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
		Name    string `json:"name"`
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

	if err := checkName(req.Name); err != nil {
		res.Error = err.Error()
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
	stmt, err := db.Prepare("UPDATE user SET name = ? WHERE user_id = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(req.Name, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	c.IndentedJSON(http.StatusCreated, res)
}

func PostUpdateEmail(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
		Email   string `json:"email"`
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

	if err := checkEmailFormat(req.Email); err != nil {
		res.Error = err.Error()
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
	stmt, err := db.Prepare("UPDATE user SET email = ?, verified = 0 WHERE user_id = ? and email <> ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(req.Email, uid, req.Email); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	c.IndentedJSON(http.StatusCreated, res)
}

func PostUpdatePassword(c *gin.Context) {
	req := struct {
		Session          string `json:"session"`
		OriginalPassword string `json:"original_password"`
		ConfirmPassword  string `json:"confirm_password"`
		NewPassword      string `json:"new_password"`
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

	// check if NewPassword == ConfirmPassword
	if req.NewPassword != req.ConfirmPassword {
		res.Error = "確認密碼與密碼不符合"
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

	// check password
	var hashedPassword, salt string
	row := db.QueryRow("SELECT password, salt FROM user WHERE user_id = ?", uid)
	if err := row.Scan(&hashedPassword, &salt); err != nil {
		res.Error = fmt.Sprintf("請重新登入 %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	if hashedPassword != util.PasswordHash(req.OriginalPassword, salt) {
		res.Error = "密碼錯誤"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// update
	salt = util.RandomString(256)
	hashedPassword = util.PasswordHash(req.NewPassword, salt)
	stmt, err := db.Prepare("UPDATE user SET salt = ?, password = ? WHERE user_id = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()

	if _, err := stmt.Exec(salt, hashedPassword, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	c.IndentedJSON(http.StatusCreated, res)
}

func PostUpdateShareGPS(c *gin.Context) {
	req := struct {
		Session    string `json:"session"`
		ShareOrNot bool   `json:"share_or_not"`
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

	stmt, err := db.Prepare("UPDATE user SET share_gps = ? WHERE user_id = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()

	if _, err := stmt.Exec(req.ShareOrNot, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	c.IndentedJSON(http.StatusCreated, res)
}

func PostUpdateGPS(c *gin.Context) {
	req := struct {
		Session string  `json:"session"`
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
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

	stmt, err := db.Prepare("UPDATE user SET last_lng = ?, last_lat = ? WHERE user_id = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	if _, err := stmt.Exec(req.Lng, req.Lat, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	c.IndentedJSON(http.StatusCreated, res)
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

	val, isLogin := session.Get(req.Session)
	if !isLogin {
		// already logout
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

	c.IndentedJSON(http.StatusOK, res)
}

func PostUpdateProfile(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
		Path    string `json:"path"`
	}{}
	res := struct {
		Error string `json:"error"`
	}{}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		c.IndentedJSON(http.StatusUnauthorized, res)
		return
	}

	db, err := sql.Open("sqlite3", config.MainDB)
	if err != nil {
		res.Error = fmt.Sprintf("sql.Open() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer db.Close()
	stmt, err := db.Prepare("UPDATE user SET profile = ? WHERE user_id = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	if _, err := stmt.Exec(req.Path, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	c.IndentedJSON(http.StatusCreated, res)
}
