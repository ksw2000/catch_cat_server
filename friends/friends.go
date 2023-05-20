package friends

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ksw2000/catch_cat_server/config"
	"github.com/ksw2000/catch_cat_server/session"
	_ "github.com/mattn/go-sqlite3"
)

func PostFriendInvite(c *gin.Context) {
	req := struct {
		Session    string `json:"session"`
		FindingUID uint64 `json:"finding_uid"`
	}{}
	res := struct {
		Error string `json:"error"`
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

	uid, isLogin := checkLogin(c, req.Session)
	if !isLogin {
		return
	}

	if uid == req.FindingUID {
		res.Error = "不可以邀請自己"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// uid cannot invite finding_id who has already invited uid or banned uid
	row := db.QueryRow("SELECT COUNT(*) FROM friend WHERE `user_id_dest` = ? and `user_id_src` = ?", uid, req.FindingUID)
	var num int
	if err := row.Scan(&num); err != nil {
		res.Error = "row.Scan() error" + err.Error()
		c.IndentedJSON(http.StatusOK, res)
		return
	} else if num > 0 {
		// user is banned by finding_id
		// or finding_id invited uid
		res.Error = "找不到 ID 或對方已邀請你"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// check if friend_id existed
	row = db.QueryRow("SELECT COUNT(*) FROM user WHERE `user_id` = ?", req.FindingUID)
	if err := row.Scan(&num); err != nil {
		res.Error = "database error row.Scan() error"
		c.IndentedJSON(http.StatusOK, res)
		return
	} else if num == 0 {
		res.Error = "找不到 ID"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// check if already friend
	row = db.QueryRow("SELECT COUNT(*) FROM friend WHERE `user_id_src` = ? and `user_id_dest` = ?", uid, req.FindingUID)
	if err := row.Scan(&num); err != nil {
		res.Error = "database error row.Scan() error"
		c.IndentedJSON(http.StatusOK, res)
		return
	} else if num > 0 {
		res.Error = "已經是好友了"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// insert
	stmt, err := db.Prepare("INSERT INTO friend(user_id_src, user_id_dest, accepted, ban) values(?, ?, ?, ?)")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	if _, err = stmt.Exec(uid, req.FindingUID, false, false); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// ok
	c.IndentedJSON(http.StatusCreated, res)
}

const (
	friendList     = 0
	invitingMeList = 1
)

func postFriends(c *gin.Context, status int) {
	req := struct {
		Session string `json:"session"`
	}{}
	type Friend struct {
		Name      string `json:"name"`
		Uid       int    `json:"uid"`
		Level     int    `json:"level"`
		LastLogin int    `json:"last_login"`
	}
	res := struct {
		Error string   `json:"error"`
		List  []Friend `json:"list"`
	}{
		List: []Friend{},
	}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := checkLogin(c, req.Session)
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

	var rows *sql.Rows

	if status == friendList {
		rows, err = db.Query(`
			SELECT 
				friend.user_id_dest as fid,
				user.name as name,
				user.last_login as last_login
			FROM friend, user
			WHERE 
				user.user_id = friend.user_id_dest and
				friend.accepted = 1 and
				friend.ban = 0     and
				friend.user_id_src = ?
		`, uid)
	} else {
		rows, err = db.Query(`
			SELECT 
				friend.user_id_src as fid,
				user.name as name,
				user.last_login as last_login
			FROM friend, user
			WHERE 
				user.user_id = friend.user_id_src and
				friend.accepted = 0 and
				friend.ban = 0   and
				friend.user_id_dest = ?
		`, uid)
	}

	if err != nil {
		res.Error = fmt.Sprintf("db.Query() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	for rows.Next() {
		friend := Friend{}
		rows.Scan(&friend.Uid, &friend.Name, &friend.LastLogin)
		res.List = append(res.List, friend)
	}

	c.IndentedJSON(http.StatusOK, res)
}

func PostFriendsList(c *gin.Context) {
	postFriends(c, friendList)
}

func PostInvitingMeList(c *gin.Context) {
	postFriends(c, invitingMeList)
}

func PostFriendDecline(c *gin.Context) {
	req := struct {
		Session   string `json:"session"`
		FriendUID uint64 `json:"friend_uid"`
	}{}
	res := struct {
		Error string `json:"error"`
	}{}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := checkLogin(c, req.Session)
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

	// if friend_id invites uid, delete the record
	stmt, err := db.Prepare("DELETE from friend WHERE user_id_src = ? and user_id_dest = ? and accepted = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(req.FriendUID, uid, false); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	c.IndentedJSON(http.StatusCreated, res)
}

func PostFriendAgree(c *gin.Context) {
	req := struct {
		Session   string `json:"session"`
		FriendUID uint64 `json:"friend_uid"`
	}{}
	res := struct {
		Error string `json:"error"`
	}{}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := checkLogin(c, req.Session)
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

	// ensure that friend_uid invite uid
	row := db.QueryRow("SELECT COUNT(*) FROM friend WHERE user_id_src = ? and user_id_dest = ? and accepted = 0", req.FriendUID, uid)
	var num int
	if err := row.Scan(&num); err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	} else if num == 0 {
		res.Error = "無此邀請"
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// update
	stmt, err := db.Prepare("INSERT INTO friend(user_id_src, user_id_dest, accepted, ban) values(?, ?, ?, ?)")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	if _, err := stmt.Exec(uid, req.FriendUID, true, false); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// insert dest->src
	stmt, err = db.Prepare("UPDATE friend SET accepted = 1 WHERE user_id_src=? and user_id_dest=?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	if _, err := stmt.Exec(req.FriendUID, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	c.IndentedJSON(http.StatusCreated, res)
}

func PostFriendDelete(c *gin.Context) {
	req := struct {
		Session   string `json:"session"`
		FriendUID uint64 `json:"friend_uid"`
	}{}
	res := struct {
		Error string `json:"error"`
	}{}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := checkLogin(c, req.Session)
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

	// delete uid -> friend_uid
	stmt, err := db.Prepare("DELETE from friend WHERE user_id_src = ? and user_id_dest = ?")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(uid, req.FriendUID); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	// delete friend_uid -> uid
	// if ban = 0
	stmt, err = db.Prepare("DELETE from friend WHERE user_id_src = ? and user_id_dest = ? and ban = 0")
	if err != nil {
		res.Error = fmt.Sprintf("db.Prepare() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(req.FriendUID, uid); err != nil {
		res.Error = fmt.Sprintf("stmt.Exec() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}

	c.IndentedJSON(http.StatusCreated, res)
}

func checkLogin(c *gin.Context, sessionID string) (uid uint64, isLogin bool) {
	val, isLogin := session.Get(sessionID)
	if !isLogin {
		c.IndentedJSON(http.StatusUnauthorized, struct {
			Error string `json:"error"`
		}{"未登入"})
		return
	}

	uid = val["uid"].(uint64)
	return
}
