package sqlconnect

import (
	"database/sql"
	"restapi/pkg/utils"
)

// GetTeacherPhotoKey returns the S3 key for a teacher's profile photo,
// or empty string if NULL. Reads from REPLICA.
func GetTeacherPhotoKey(id int) (string, error) {
	db, err := ConnectReadDb("schooldb")
	if err != nil {
		return "", utils.ErrorHandler(err, "Failed to connect to database")
	}
	var key sql.NullString
	err = db.QueryRow("SELECT photo_s3_key FROM teachers WHERE id = ?", id).Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", utils.ErrorHandler(err, "Teacher not found")
		}
		return "", utils.ErrorHandler(err, "Failed to query teacher photo")
	}
	if !key.Valid {
		return "", nil
	}
	return key.String, nil
}

// SetTeacherPhotoKey writes (or clears via empty string) the photo_s3_key.
// Goes to PRIMARY.
func SetTeacherPhotoKey(id int, key string) error {
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
	res, err := db.Exec("UPDATE teachers SET photo_s3_key = ? WHERE id = ?", arg, id)
	if err != nil {
		return utils.ErrorHandler(err, "Failed to update teacher photo")
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return utils.ErrorHandler(sql.ErrNoRows, "Teacher not found")
	}
	return nil
}
