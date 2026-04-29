package sqlconnect

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"restapi/internal/models"
	"restapi/pkg/utils"
	"strconv"

)

func GetTeachersDBHandler(teachers []models.Teacher, r *http.Request) ([]models.Teacher, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	query := "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE 1=1"
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
		fmt.Printf("Error querying teachers: %v\n", err)
		// http.Error(w, "Failed to query teachers", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to query teachers")
	}
	defer rows.Close()
	// teachersList := make([]models.Teacher, 0)

	for rows.Next() {
		var teacher models.Teacher
		err := rows.Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
		if err != nil {
			fmt.Printf("Error scanning teacher row: %v\n", err)
			// http.Error(w, "Failed to scan teacher", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to scan teacher")
		}
		teachers = append(teachers, teacher)
	}
	return teachers, nil
}

func GetTeacherByID(id int) (models.Teacher, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err != nil {
		if err == sql.ErrNoRows {
			// http.Error(w, "Teacher not found", http.StatusNotFound)
			return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
		}
		// http.Error(w, "Failed to query teacher", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to query teacher")
	}
	return teacher, nil
}
func isValidSortOrder(order string) bool {
	return order == "asc" || order == "desc"
}

func isValidSortField(field string) bool {
	validFields := map[string]bool{
		"first_name": true,
		"last_name":  true,
		"email":      true,
		"class":      true,
		"subject":    true,
	}
	return validFields[field]

}



func AddTeachersDBHandler(newTeachers []models.Teacher) ([]models.Teacher, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	// stmnt, err := db.Prepare("INSERT INTO teachers (first_name, last_name, email, class, subject) VALUES (?, ?, ?, ?, ?)")
	stmnt, err := db.Prepare(utils.GenerateInsertQuery("teachers", models.Teacher{}))

	if err != nil {
		// http.Error(w, "Failed to prepare statement", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to prepare statement")
	}
	defer stmnt.Close()

	addedTeachers := make([]models.Teacher, len(newTeachers))
	for i, newTeacher := range newTeachers {
		// res, err := stmnt.Exec(newTeacher.FirstName, newTeacher.LastName, newTeacher.Email, newTeacher.Class, newTeacher.Subject)
		values:= utils.GetStructValues(newTeacher)
		res, err := stmnt.Exec(values...)

		if err != nil {
			// http.Error(w, "Failed to insert teacher", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to insert teacher")
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			// http.Error(w, "Failed to retrieve last insert id", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to retrieve last insert id")
		}
		newTeacher.ID = int(lastId)
		addedTeachers[i] = newTeacher
	}
	return addedTeachers, nil
}


func UpdateTeacher(id int, updatedTeacher models.Teacher) (models.Teacher, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
	} else if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve teacher", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to retrieve teacher")
	}
	updatedTeacher.ID = existingTeacher.ID
	_, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", updatedTeacher.FirstName, updatedTeacher.LastName, updatedTeacher.Email, updatedTeacher.Class, updatedTeacher.Subject, updatedTeacher.ID)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to update teacher", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to update teacher")
	}
	return updatedTeacher, nil
}

func PatchTeachers(updates []map[string]interface{}) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return utils.ErrorHandler(err,"Failed to connect to database")
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
			// http.Error(w, "Invalid or missing teacher ID", http.StatusBadRequest)
			return utils.ErrorHandler(err, "Invalid or missing teacher ID")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			tx.Rollback()
			// http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
			return utils.ErrorHandler(err, "Invalid teacher ID")
		}
		var teacherFromDb models.Teacher
		err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", int(id)).Scan(&teacherFromDb.ID, &teacherFromDb.FirstName, &teacherFromDb.LastName, &teacherFromDb.Email, &teacherFromDb.Class, &teacherFromDb.Subject)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {

				// http.Error(w, fmt.Sprintf("Teacher with ID %d not found", id), http.StatusNotFound)
				return utils.ErrorHandler(err, "Teacher not found")
			}
			log.Println(err)
			// http.Error(w, "Failed to retrieve teacher", http.StatusInternalServerError)
			return utils.ErrorHandler(err, "Failed to retrieve teacher")
		}
		// apply updates to teacherFromDb using reflect package

		teacherVal := reflect.ValueOf(&teacherFromDb).Elem()
		teacherType := teacherVal.Type()

		for k, v := range update {
			if k == "id" {
				continue
			}
			for i := 0; i < teacherVal.NumField(); i++ {
				field := teacherType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := teacherVal.Field(i)
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
		_, err = tx.Exec("UPDATE teachers SET first_name=?,last_name=?, email=?, class=?, subject=? WHERE id=?", teacherFromDb.FirstName, teacherFromDb.LastName, teacherFromDb.Email, teacherFromDb.Class, teacherFromDb.Subject, teacherFromDb.ID)
		if err != nil {
			tx.Rollback()
			log.Println(err)
			// http.Error(w, "Failed to update teacher", http.StatusInternalServerError)
			return utils.ErrorHandler(err, "Failed to update teacher")
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

func PatchOneTeacher(id int, updates map[string]interface{}) (models.Teacher, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
	} else if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve teacher", http.StatusInternalServerError)
		return models.Teacher{

			// for k,v := range updates{
			// 	switch k {
			// 	case "first_name":
			// 		existingTeacher.FirstName = v.(string)
			// 	case "last_name":
			// 		existingTeacher.LastName = v.(string)
			// 	case "email":
			// 		existingTeacher.Email = v.(string)
			// 	case "class":
			// 		existingTeacher.Class = v.(string)
			// 	case "subject":
			// 		existingTeacher.Subject = v.(string)
			// 	}
			// }
		}, utils.ErrorHandler(err,"Failed to retrieve teacher")
	}

	// Apply updates using reflect package

	teacherVal := reflect.ValueOf(&existingTeacher).Elem()
	teacherType := teacherVal.Type()

	for k, v := range updates {
		for i := 0; i < teacherVal.NumField(); i++ {
			fmt.Println("k from reflect mechanism", k)
			field := teacherType.Field(i)
			fmt.Println(field.Tag.Get("json"))
			if field.Tag.Get("json") == k+",omitempty" {
				if teacherVal.Field(i).CanSet() {
					teacherVal.Field(i).Set(reflect.ValueOf(v).Convert(teacherVal.Field(i).Type()))
				}
			}

		}
	}

	_, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", existingTeacher.FirstName, existingTeacher.LastName, existingTeacher.Email, existingTeacher.Class, existingTeacher.Subject, existingTeacher.ID)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to update teacher", http.StatusInternalServerError)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to update teacher")
	}
	return existingTeacher, nil
}

func DeleteOneTeacher(id int) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	res, err := db.Exec("DELETE FROM teachers WHERE id = ?", id)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to delete teacher")
	}
	fmt.Println(res.RowsAffected())
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve affected rows", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to retrieve affected rows")
	}
	if rowsAffected == 0 {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return utils.ErrorHandler(err, "Teacher not found")
	}
	return nil
}

func DeleteTeachers(ids []int) ([]int, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to start transaction")
	}
	stmnt, err := tx.Prepare("DELETE FROM teachers WHERE id = ?")
	if err != nil {
		log.Println(err)
		tx.Rollback()
		// http.Error(w, "Failed to prepare delete statement", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to prepare delete statement")
	}
	defer stmnt.Close()

	deletedIds := []int{}
	for _, id := range ids {
		res, err := stmnt.Exec(id)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			// http.Error(w, "Failed to delete teacher with ID "+strconv.Itoa(id), http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to delete teacher with ID "+strconv.Itoa(id))
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Println(err)
			tx.Rollback()
			// http.Error(w, "Failed to retrieve affected rows for ID "+strconv.Itoa(id), http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to retrieve affected rows for ID "+strconv.Itoa(id))
		}
		if rowsAffected > 0 {
			deletedIds = append(deletedIds, id)
		}
		if rowsAffected < 1 {
			tx.Rollback()
			// http.Error(w, "Teacher with ID "+strconv.Itoa(id)+" not found, does not exist!!", http.StatusNotFound)
			return nil, utils.ErrorHandler(err, "Teacher with ID "+strconv.Itoa(id)+" not found, does not exist!!")
		}
	}

	// Commmit
	err = tx.Commit()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to commit transaction")
	}
	if len(deletedIds) < 1 {
		// http.Error(w, "ID's do not exist and No teachers were deleted", http.StatusBadRequest)
		return nil, utils.ErrorHandler(err, "ID's do not exist and No teachers were deleted")
	}
	return deletedIds, nil
}
func GetStudentsByTeacherIdromDb(teacherid string, w http.ResponseWriter, students []models.Student) ([]models.Student, error) {
	db, err := ConnectDb("schooldb")

	if err != nil {
		log.Println(err)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	query := "SELECT id, first_name, last_name, email, class FROM students WHERE class = (SELECT class FROM teachers WHERE id = ?)"
	rows, err := db.Query(query, teacherid)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to retrieve students for the teacher", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to retrieve students for the teacher")

	}
	defer rows.Close()
	for rows.Next() {
		var student models.Student
		err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to scan student data", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to scan student data")
		}
		students = append(students, student)
	}
	err = rows.Err()
	if err != nil {
		log.Println(err)
		http.Error(w, "Error occurred while retrieving students", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Error occurred while retrieving students")
	}
	return students, nil
}


func GetStudentCountByTeacherIdFromDb(teacherid string) (int,error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		log.Println(err)
		return 0, utils.ErrorHandler(err, "Failed to connect to database")
	}
	
	// shared pool — do not close

	query := "SELECT COUNT(*) FROM students WHERE class = (SELECT class FROM teachers WHERE id = ?)"
	var studentCount int
	err = db.QueryRow(query, teacherid).Scan(&studentCount)
	if err != nil {
		log.Println(err)

		return 0, utils.ErrorHandler(err, "Failed to retrieve student count")
	}
	return studentCount, nil
}
