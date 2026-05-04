package sqlconnect

import (
	"database/sql"
	"restapi/pkg/utils"
)

// GetExecPhotoKey returns the S3 key for an exec's profile photo,
// or empty string if NULL. Reads from REPLICA.
func GetExecPhotoKey(id int) (string, error) {
	db, err := ConnectReadDb("schooldb")
	if err != nil {
		return "", utils.ErrorHandler(err, "Failed to connect to database")
	}
	var key sql.NullString
	err = db.QueryRow("SELECT photo_s3_key FROM execs WHERE id = ?", id).Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", utils.ErrorHandler(err, "Exec not found")
		}
		return "", utils.ErrorHandler(err, "Failed to query exec photo")
	}
	if !key.Valid {
		return "", nil
	}
	return key.String, nil
}

// SetExecPhotoKey writes (or clears via empty string) the photo_s3_key.
// Goes to PRIMARY.
func SetExecPhotoKey(id int, key string) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		return utils.ErrorHandler(err, "Failed to connect to database")
	}
	var arg interface{}
	if key == "" {
		arg = nil
	} else {
		arg = key
	}
	res, err := db.Exec("UPDATE execs SET photo_s3_key = ? WHERE id = ?", arg, id)
	if err != nil {
		return utils.ErrorHandler(err, "Failed to update exec photo")
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return utils.ErrorHandler(sql.ErrNoRows, "Exec not found")
	}
	return nil
}
