package sqlconnect

import (
	"database/sql"
	"restapi/pkg/utils"
)

// GetStudentPhotoKey returns the S3 key for a student's profile photo,
// or empty string + nil if they have no photo.
// Returns sql.ErrNoRows wrapped if the student doesn't exist.
//
// Reads from REPLICA — staleness here is fine. If we just uploaded a photo
// and the replica hasn't caught up yet (~ms), worst case is a brief 404.
func GetStudentPhotoKey(id int) (string, error) {
	db, err := ConnectReadDb("schooldb")
	if err != nil {
		return "", utils.ErrorHandler(err, "Failed to connect to database")
	}
	var key sql.NullString
	err = db.QueryRow("SELECT photo_s3_key FROM students WHERE id = ?", id).Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", utils.ErrorHandler(err, "Student not found")
		}
		return "", utils.ErrorHandler(err, "Failed to query student photo")
	}
	if !key.Valid {
		return "", nil // student exists, photo column is NULL
	}
	return key.String, nil
}

// SetStudentPhotoKey writes (or clears) the photo_s3_key on a student row.
// Pass empty string to clear. Goes to PRIMARY (write).
func SetStudentPhotoKey(id int, key string) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		return utils.ErrorHandler(err, "Failed to connect to database")
	}
	var arg interface{}
	if key == "" {
		arg = nil // store SQL NULL, not an empty string
	} else {
		arg = key
	}
	res, err := db.Exec("UPDATE students SET photo_s3_key = ? WHERE id = ?", arg, id)
	if err != nil {
		return utils.ErrorHandler(err, "Failed to update student photo")
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return utils.ErrorHandler(sql.ErrNoRows, "Student not found")
	}
	return nil
}
