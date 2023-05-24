package util

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func PasswordHash(pwd string, salt string) string {
	pwd += salt
	return fmt.Sprintf("%x", sha256.Sum256([]byte(pwd)))
}

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GetScoreAndLevel(db *sql.DB, uid uint64) (cats int, score int, level int) {
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
