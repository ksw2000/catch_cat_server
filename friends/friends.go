package friends

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ksw2000/catch_cat_server/session"
	"github.com/ksw2000/catch_cat_server/util"
	_ "github.com/mattn/go-sqlite3"
)

type Friend struct {
	Name       string  `json:"name"`
	Uid        uint64  `json:"uid"`
	Profile    string  `json:"profile"`
	Level      int     `json:"level"`
	Score      int     `json:"score"`
	Cats       int     `json:"cats"`
	LastLogin  int     `json:"last_login"`
	ThemeScore int     `json:"theme_score"` // optional
	ThemeCats  int     `json:"theme_cats"`  // optional
	Lat        float64 `json:"lat"`         // optional
	Lng        float64 `json:"lng"`         // optional
}

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

	db := util.OpenDB()

	uid, isLogin := session.CheckLogin(c, req.Session)
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
	defer stmt.Close()
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
	friendPosition = 2
	themeRank      = 3
)

func postFriends(c *gin.Context, status int) {
	req := struct {
		Session string `json:"session"`
		ThemeID int    `json:"theme_id"`
	}{}

	res := struct {
		Error string   `json:"error"`
		List  []Friend `json:"list"`
	}{
		List: []Friend{},
	}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		return
	}

	db := util.OpenDB()

	var err error
	var rows *sql.Rows

	if status == friendList {
		rows, err = db.Query(`
			SELECT 
				friend.user_id_dest as fid,
				user.name,
				user.profile,
				user.last_login
			FROM friend, user
			WHERE 
				user.user_id = friend.user_id_dest and
				friend.accepted = 1 and
				friend.ban = 0     and
				friend.user_id_src = ?
		`, uid)
	} else if status == invitingMeList {
		rows, err = db.Query(`
			SELECT 
				friend.user_id_src as fid,
				user.name,
				user.profile,
				user.last_login
			FROM friend, user
			WHERE 
				user.user_id = friend.user_id_src and
				friend.accepted = 0 and
				friend.ban = 0   and
				friend.user_id_dest = ?
		`, uid)
	} else if status == friendPosition {
		// rows, err = db.Query(`
		// 	SELECT
		// 		friend.user_id_dest as fid,
		// 		user.name,
		// 		user.profile,
		// 		user.last_login,
		// 		user.last_lat,
		// 		user.last_lng
		// 	FROM friend, user
		// 	WHERE
		// 		user.user_id = friend.user_id_dest and
		// 		friend.accepted = 1 and
		// 		friend.ban = 0      and
		// 		user.share_gps = 1  and
		// 		friend.user_id_src = ?
		// `, uid)
		rows, err = db.Query(`
			SELECT
				ta.*,
				SUM(tb.weight) as score,
				COUNT(tb.cat_id) as cats
			FROM
				(
					SELECT
						user.user_id,
						user.name,
						user.profile,
						user.last_login,
						user.last_lat,
						user.last_lng
						FROM user, friend
					WHERE 
						friend.user_id_src = ? and
						friend.user_id_dest = user.user_id and 
						friend.accepted = 1 AND
						friend.ban = 0
				) as ta
			LEFT JOIN
				(
					SELECT *
					FROM user_cat, cat, cat_kind
					WHERE
						user_cat.cat_id = cat.cat_id and
						cat.cat_kind_id = cat_kind.cat_kind_id and
						cat.theme_id = ?
				) as tb
				ON
					tb.user_id = ta.user_id
			GROUP BY ta.user_id
			ORDER BY score DESC`, uid, req.ThemeID)
	} else if status == themeRank {
		rows, err = db.Query(`
			SELECT
				ta.*,
				SUM(tb.weight) as score,
				COUNT(tb.cat_id) as cats
			FROM
				(
					SELECT
						user.user_id,
						user.name,
						user.profile,
						user.last_login
						FROM user, friend
					WHERE 
						(
							friend.user_id_src = ? and
							friend.user_id_dest = user.user_id and 
							friend.accepted = 1 AND
							friend.ban = 0
						) UNION
					SELECT
						user.user_id,
						user.name,
						user.profile,
						user.last_login
					FROM user
					WHERE user.user_id = ?
				) as ta
			LEFT JOIN
				(
					SELECT *
					FROM user_cat, cat, cat_kind
					WHERE
						user_cat.cat_id = cat.cat_id and
						cat.cat_kind_id = cat_kind.cat_kind_id and
						cat.theme_id = ?
				) as tb
				ON
					tb.user_id = ta.user_id
			GROUP BY ta.user_id
			ORDER BY score DESC`, uid, uid, req.ThemeID)
	}

	if err != nil {
		res.Error = fmt.Sprintf("db.Query() error %v", err)
		c.IndentedJSON(http.StatusOK, res)
		return
	}
	defer rows.Close()

	for rows.Next() {
		friend := Friend{}
		if status == friendPosition {
			rows.Scan(&friend.Uid, &friend.Name, &friend.Profile, &friend.LastLogin, &friend.Lat, &friend.Lng, &friend.ThemeScore, &friend.ThemeCats)
		} else if status == themeRank {
			rows.Scan(&friend.Uid, &friend.Name, &friend.Profile, &friend.LastLogin, &friend.ThemeScore, &friend.ThemeCats)
		} else {
			rows.Scan(&friend.Uid, &friend.Name, &friend.Profile, &friend.LastLogin)
		}
		friend.Cats, friend.Score, friend.Level = util.GetScoreAndLevel(db, friend.Uid)
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

func PostFriendsPosition(c *gin.Context) {
	postFriends(c, friendPosition)
}

func PostFriendRankAtTheme(c *gin.Context) {
	postFriends(c, themeRank)
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

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		return
	}

	db := util.OpenDB()

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

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		return
	}

	db := util.OpenDB()

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
	defer stmt.Close()
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
	defer stmt.Close()
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

	uid, isLogin := session.CheckLogin(c, req.Session)
	if !isLogin {
		return
	}

	db := util.OpenDB()

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
