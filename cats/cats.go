package cats

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ksw2000/catch_cat_server/config"
	"github.com/ksw2000/catch_cat_server/session"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func PostTheme(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
		ThemeID uint64 `json:"theme_id"`
	}{}
	type Cat struct {
		CatID       uint64  `json:"cat_id"`
		CatKindID   uint64  `json:"cat_kind_id"`
		Weight      int     `json:"weight"`
		Lng         float64 `json:"lng"`
		Lat         float64 `json:"lat"`
		Thumbnail   string  `json:"thumbnail"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		IsCaught    bool    `json:"is_caught"`
	}
	res := struct {
		Error   string `json:"error"`
		CatList []Cat  `json:"cat_list"`
	}{
		CatList: []Cat{},
	}

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

	rows, err := db.Query(`
		SELECT cat.cat_id, cat.cat_kind_id, cat.lng, cat.lat, 
		       cat_kind.thumbnail, cat_kind.weight,
			   cat_kind.description, cat_kind.name
		FROM cat, cat_kind 
		WHERE theme_id = ? and cat.cat_kind_id = cat_kind.cat_kind_id`, req.ThemeID)
	if err != nil {
		res.Error = fmt.Sprintf("db.Query() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	for rows.Next() {
		cat := Cat{}
		rows.Scan(&cat.CatID, &cat.CatKindID, &cat.Lng, &cat.Lat, &cat.Thumbnail, &cat.Weight, &cat.Description, &cat.Name)
		cat.IsCaught = isCaught(db, uid, cat.CatID)
		res.CatList = append(res.CatList, cat)
	}

	c.IndentedJSON(http.StatusOK, res)
}

func isCaught(db *sql.DB, uid uint64, catID uint64) bool {
	var num = 0
	row := db.QueryRow("SELECT COUNT(*) FROM user_cat WHERE user_id = ? and cat_id = ?", uid, catID)
	row.Scan(&num)
	return num > 0
}

func PostCatching(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
		CatID   uint64 `json:"cat_id"`
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

	// insert
	stmt, err := db.Prepare("INSERT INTO user_cat(user_id, cat_id, timing) values(?, ?, ?)")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	now := time.Now().Unix()
	if _, err = stmt.Exec(uid, req.CatID, now); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	// ok
	c.IndentedJSON(http.StatusCreated, res)
}
