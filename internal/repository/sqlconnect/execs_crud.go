package sqlconnect

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"restapi/internal/models"
	"restapi/pkg/utils"

	"github.com/go-mail/mail/v2"
)

func GetExecsDBHandler(execs []models.Exec, r *http.Request) ([]models.Exec, error) {
	db, err := ConnectReadDb("schooldb") // SELECT — replica is fine
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	query := "SELECT id, first_name, last_name, email,username, user_created_at,inactive_status,role FROM execs WHERE 1=1"
	var args []interface{}
	// if firstName != "" {
	// 	query += " AND first_name = ?"
	// 	args = append(args, firstName)
	// }
	// if lastName != "" {
	// 	query += " AND last_name = ?"
	// 	args = append(args, lastName)
	// }
	query, args = utils.AddFilters(r, query, args)

	query = utils.AddSorting(r, query)
	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying Exec: %v\n", err)
		// http.Error(w, "Failed to query execs", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to query execs")
	}
	defer rows.Close()
	// execsList := make([]models.Exec, 0)

	for rows.Next() {
		var exec models.Exec
		err := rows.Scan(&exec.ID, &exec.FirstName, &exec.LastName, &exec.Email,&exec.Username, &exec.UserCreatedAt, &exec.InactiveStatus, &exec.Role)
		if err != nil {
			fmt.Printf("Error scanning exec row: %v\n", err)
			// http.Error(w, "Failed to scan exec", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to scan exec")
		}
		execs = append(execs, exec)
	}
	return execs, nil
}

func GetExecByID(id int) (models.Exec, error) {
	db, err := ConnectReadDb("schooldb") // SELECT — replica is fine
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Exec{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	var exec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username, user_created_at, inactive_status, role  FROM execs WHERE id = ?", id).Scan(&exec.ID, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.UserCreatedAt, &exec.InactiveStatus, &exec.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			// http.Error(w, "Student not found", http.StatusNotFound)
			return models.Exec{}, utils.ErrorHandler(err, "Exec not found")
		}
		// http.Error(w, "Failed to query student", http.StatusInternalServerError)
		return models.Exec{}, utils.ErrorHandler(err, "Failed to query exec")
	}
	return exec, nil
}

func AddExecsDBHandler(newExecs []models.Exec) ([]models.Exec, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	// stmnt, err := db.Prepare("INSERT INTO execs (first_name, last_name, email, username) VALUES (?, ?, ?, ?)")
	stmnt, err := db.Prepare(utils.GenerateInsertQuery("execs", models.Exec{}))

	if err != nil {
		// http.Error(w, "Failed to prepare statement", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to prepare statement")
	}
	defer stmnt.Close()

	addedExecs := make([]models.Exec, len(newExecs))
	for i, newExec := range newExecs {
		// res, err := stmnt.Exec(newExec.FirstName, newExec.LastName, newExec.Email, newExec.Username)
		newExec.Password, err = utils.HashPassword(newExec.Password)
		if err !=nil {

			return nil,utils.ErrorHandler(err,"error adding exec into db")
		}


		values := utils.GetStructValues(newExec)
		res, err := stmnt.Exec(values...)

		if err != nil {
			// http.Error(w, "Failed to insert exec", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to insert exec")
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			// http.Error(w, "Failed to retrieve last insert id", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to retrieve last insert id")
		}
		newExec.ID = int(lastId)
		addedExecs[i] = newExec
	}
	return addedExecs, nil
}



func PatchExecs(updates []map[string]interface{}) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to start transaction")
	}
	for _, update := range updates {
		idStr, ok := update["id"].(string)
		if !ok {
			tx.Rollback()
			// http.Error(w, "Invalid or missing student ID", http.StatusBadRequest)
			return utils.ErrorHandler(err, "Invalid or missing student ID")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			tx.Rollback()
			// http.Error(w, "Invalid student ID", http.StatusBadRequest)
			return utils.ErrorHandler(err, "Invalid student ID")
		}
		var ExecFromDb models.Exec
		err = db.QueryRow("SELECT id, first_name, last_name, email, username FROM execs WHERE id = ?", int(id)).Scan(&ExecFromDb.ID, &ExecFromDb.FirstName, &ExecFromDb.LastName, &ExecFromDb.Email, &ExecFromDb.Username)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {

				// http.Error(w, fmt.Sprintf("Student with ID %d not found", id), http.StatusNotFound)
				return utils.ErrorHandler(err, "Exec not found")
			}
			log.Println(err)
			// http.Error(w, "Failed to retrieve student", http.StatusInternalServerError)
			return utils.ErrorHandler(err, "Failed to retrieve exec")
		}
		// apply updates to studentFromDb using reflect package

		execVal := reflect.ValueOf(&ExecFromDb).Elem()
		execType := execVal.Type()

		for k, v := range update {
			if k == "id" {
				continue
			}
			for i := 0; i < execVal.NumField(); i++ {
				field := execType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := execVal.Field(i)
					if fieldVal.CanSet() {
						val := reflect.ValueOf(v)
						if val.Type().ConvertibleTo(fieldVal.Type()) {
							fieldVal.Set(val.Convert(fieldVal.Type()))
						} else {
							tx.Rollback()
							log.Printf("Cannot convert %v for %v", val.Type(), fieldVal.Type())
							return err
						}
					}
					break

				}
			}

		}
		_, err = tx.Exec("UPDATE execs SET first_name=?,last_name=?, email=?, username=? WHERE id=?", ExecFromDb.FirstName, ExecFromDb.LastName, ExecFromDb.Email, ExecFromDb.Username,  ExecFromDb.ID)
		if err != nil {
			tx.Rollback()
			log.Println(err)
			// http.Error(w, "Failed to update exec", http.StatusInternalServerError)
			return utils.ErrorHandler(err, "Failed to update exec")
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to commit transaction")
	}
	return nil
}

func PatchOneExec(id int, updates map[string]interface{}) (models.Exec, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Exec{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	var existingExec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username FROM execs WHERE id = ?", id).Scan(&existingExec.ID, &existingExec.FirstName, &existingExec.LastName, &existingExec.Email, &existingExec.Username)
	if err == sql.ErrNoRows {
		// http.Error(w, "Student not found", http.StatusNotFound)
		return models.Exec{}, utils.ErrorHandler(err, "Exec not found")
	} else if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve student", http.StatusInternalServerError)
		return models.Exec{

			// for k,v := range updates{
			// 	switch k {
			// 	case "first_name":
			// 		existingStudent.FirstName = v.(string)
			// 	case "last_name":
			// 		existingStudent.LastName = v.(string)
			// 	case "email":
			// 		existingStudent.Email = v.(string)
			// 	case "class":
			// 		existingStudent.Class = v.(string)
			// 	}
			// }
		}, utils.ErrorHandler(err, "Failed to retrieve exec")
	}

	// Apply updates using reflect package

	execVal := reflect.ValueOf(&existingExec).Elem()
	execType := execVal.Type()

	for k, v := range updates {
		for i := 0; i < execVal.NumField(); i++ {
			fmt.Println("k from reflect mechanism", k)
			field := execType.Field(i)
			fmt.Println(field.Tag.Get("json"))
			if field.Tag.Get("json") == k+",omitempty" {
				if execVal.Field(i).CanSet() {
					execVal.Field(i).Set(reflect.ValueOf(v).Convert(execVal.Field(i).Type()))
				}
			}

		}
	}

	_, err = db.Exec("UPDATE execs SET first_name = ?, last_name = ?, email = ?, username = ?, user_created_at = ?, inactive_status = ?, role = ? WHERE id = ?", existingExec.FirstName, existingExec.LastName, existingExec.Email, existingExec.Username, existingExec.UserCreatedAt, existingExec.InactiveStatus, existingExec.Role, existingExec.ID)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to update exec", http.StatusInternalServerError)
		return models.Exec{}, utils.ErrorHandler(err, "Failed to update exec")
	}
	return existingExec, nil
}

func DeleteOneExec(id int) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	res, err := db.Exec("DELETE FROM execs WHERE id = ?", id)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to delete exec", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to delete exec")
	}
	fmt.Println(res.RowsAffected())
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve affected rows", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to retrieve affected rows")
	}
	if rowsAffected == 0 {
		// http.Error(w, "Exec not found", http.StatusNotFound)
		return utils.ErrorHandler(err, "Exec not found")
	}
	return nil
}

func GetUserByUsername(username string) (*models.Exec, error) {
	// Login auth — explicitly use PRIMARY (Write pool).
	// New signups / password changes must work immediately;
	// using the replica here would cause race conditions where a fresh user
	// appears to "not exist" for a few seconds after creation.
	db, err := ConnectDb("schooldb")
	if err != nil {
		log.Println(err)
		return  nil,utils.ErrorHandler(err, "Database connection error")
		// http.Error(w, "error connecting to database", http.StatusUnauthorized)
		// return nil, true
	}
	// shared pool — do not close
	user := &models.Exec{}
	err = db.QueryRow(`SELECT id, first_name, last_name, email, username, password,inactive_status,role FROM execs WHERE username = ?`, username).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Username, &user.Password, &user.InactiveStatus, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			 return nil,utils.ErrorHandler(err, "user not found")
			// http.Error(w, "user not found", http.StatusUnauthorized)
			// return nil, true
		}
		// log.Println(err)
		return nil,utils.ErrorHandler(err, "Database query error")	
	}
	return user, nil
}

func UpdatePasswordinDB(userId int, currentPassword, newPassword string)(bool, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		return false, utils.ErrorHandler(err, "Database connection error")
		
	}
	// shared pool — do not close

	var username string
	var userPassword string
	var userRole string

	err = db.QueryRow("SELECT username, password, role FROM execs WHERE id = ?", userId).Scan(&username, &userPassword, &userRole)
	if err != nil {
		return false,utils.ErrorHandler(err, "user not found")
		
	}

	err = utils.VerifyPassword(currentPassword, userPassword)
	if err != nil {
		return false,utils.ErrorHandler(err, "The pass word you enered does not match the current pass word on file")
		
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return false,utils.ErrorHandler(err, "internal server error")
		
	}
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE execs SET password=?, password_changed_at=? WHERE id =?", hashedPassword, currentTime, userId)
	if err != nil {
		return false,utils.ErrorHandler(err, "failed to update password")
	
	}

	// 	token,err := utils.SignToken(userId,username,userRole)
	// if err!=nil{
	// 	utils.ErrorHandler(err,"password udpated, could not create token")
	// 	return
	// }
	return true,nil
}

func ForgotPasswordDBHandler(emailId string) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		return utils.ErrorHandler(err, "Internal error")
		
	}
	// shared pool — do not close

	var exec models.Exec
	err = db.QueryRow("SELECT id FROM execs WHERE  email=?",emailId).Scan(&exec.ID)
	if err != nil {
		return  utils.ErrorHandler(err, "User not ofound")
		

	}

	duration, err := strconv.Atoi(os.Getenv("RESET_TOKEN_EXP_DURATION"))
	if err != nil {
		return  utils.ErrorHandler(err, "Failed to send password reset email")
		
	}
	mins := time.Duration(duration)

	expiry := time.Now().Add(mins * time.Minute).Format("2006-01-02 15:04:05")

	tokenBytes := make([]byte, 32)

	_, err = rand.Read(tokenBytes)
	if err != nil {
		return  utils.ErrorHandler(err, "Failed to send password reset mail")
		
	}

	log.Println("TokenBYtes:", tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	log.Println("Token:", token)

	hashedToken := sha256.Sum256(tokenBytes)
	log.Println("hashedToken:", hashedToken)

	hashedTokenString := hex.EncodeToString(hashedToken[:])
	_, err = db.Exec("UPDATE execs SET password_reset_code = ?, password_reset_code_expires = ? WHERE id = ?", hashedTokenString, expiry, exec.ID)
	if err != nil {
		return  utils.ErrorHandler(err, "Failed to send password reset email")
		
	}

	// Email
	resetURL := fmt.Sprintf("https://localhost:3000/execs/resetpassword/reset/%s", token)
	message := fmt.Sprintf("Forgot your password? Reset your password using the following link:\n%s\nIf you did no request a password reset, please ignore this email This link is only valid for %d minutes.", resetURL, int(mins))

	m := mail.NewMessage()
	m.SetHeader("From", "schooladmin@school.com")
	m.SetHeader("To", emailId)
	m.SetHeader("Subject", "Your password reset link")
	m.SetBody("text/plain", message)

	d := mail.NewDialer("localhost", 1025, "", "")
	err = d.DialAndSend(m)
	if err != nil {
		return  utils.ErrorHandler(err, "Failed to send password reset email")
		
	}
	return nil
}


func ResetPasswordDBHandler(token string,NewPassword string) error {

	bytes,err := hex.DecodeString(token)
	if err !=nil{
		return utils.ErrorHandler(err, "Internal error")
	}

	hashedToken := sha256.Sum256(bytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	db, err := ConnectDb("schooldb")
	if err != nil {
		
		

	}
	// shared pool — do not close
	var user models.Exec

	query := "SELECT id,email FROM execs WHERE password_reset_code = ? AND password_reset_code_expires > ?"
	err = db.QueryRow(query, hashedTokenString, time.Now().Format(time.RFC3339)).Scan(&user.ID, &user.Email)
	if err != nil {
		
		return utils.ErrorHandler(err, "Invalid or expired reset code")
	}

	hashedPassword, err := utils.HashPassword(NewPassword)
	if err != nil {
		
		return utils.ErrorHandler(err, "Internal error")
	}

	updateQuery := "UPDATE execs SET password =?, password_reset_code=NULL, password_reset_code_expires=NULL, password_changed_at=? WHERE id =?"
	_, err = db.Exec(updateQuery, hashedPassword, time.Now().Format(time.RFC3339), user.ID)
	if err != nil {
		
		return utils.ErrorHandler(err, "Internal error")
	}
	return nil
}
