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

type CatKind struct {
	CatKindID   uint64 `json:"cat_kind_id"`
	Weight      int    `json:"weight"`
	Thumbnail   string `json:"thumbnail"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func PostTheme(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
		ThemeID uint64 `json:"theme_id"`
	}{}
	type Cat struct {
		CatID    uint64  `json:"cat_id"`
		Lng      float64 `json:"lng"`
		Lat      float64 `json:"lat"`
		IsCaught bool    `json:"is_caught"`
		CatKind
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
	defer rows.Close()
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
	defer stmt.Close()
	now := time.Now().Unix()
	if _, err = stmt.Exec(uid, req.CatID, now); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	// ok
	c.IndentedJSON(http.StatusCreated, res)
}

func PostCaughtKind(c *gin.Context) {
	req := struct {
		Session string `json:"session"`
	}{}
	type CatKindCaught struct {
		IsCaught bool `json:"is_caught"`
		CatKind
	}
	res := struct {
		Error string          `json:"error"`
		List  []CatKindCaught `json:"list"`
	}{
		List: []CatKindCaught{},
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

	rows, err := db.Query(`SELECT 
		cat_kind.cat_kind_id,
		cat_kind.name,
		cat_kind.description,
		cat_kind.weight,
		cat_kind.thumbnail,
		user_cat.user_id
	FROM
		cat_kind
	JOIN
		cat
	on cat.cat_kind_id = cat_kind.cat_kind_id
	LEFT JOIN
		user_cat
	on cat.cat_id = user_cat.cat_id and user_cat.user_id = ?
	GROUP BY cat_kind.cat_kind_id
	ORDER BY cat_kind.cat_kind_id ASC`, uid)
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer rows.Close()
	for rows.Next() {
		scanner := CatKindCaught{}
		var id uint64
		rows.Scan(&scanner.CatKindID, &scanner.Name, &scanner.Description, &scanner.Weight, &scanner.Thumbnail, &id)
		scanner.IsCaught = id != 0
		res.List = append(res.List, scanner)
	}

	// ok
	c.IndentedJSON(http.StatusOK, res)
}
