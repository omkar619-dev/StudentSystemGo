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
	"strings"

)

func GetStudentsDBHandler(students []models.Student, r *http.Request,limit,page int) ([]models.Student,int, error) {
	db, err := ConnectReadDb("schooldb") // SELECT — replica is fine
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, 0,utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	query := "SELECT id, first_name, last_name, email, class FROM students WHERE 1=1"
	var args []interface{}
	query, args = utils.AddFilters(r, query, args)

	query = utils.AddSorting(r, query)

	offset := (page - 1) * limit
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying students: %v\n", err)
		// http.Error(w, "Failed to query students", http.StatusInternalServerError)
		return nil, 0,utils.ErrorHandler(err, "Failed to query students")
	}
	defer rows.Close()
	// studentsList := make([]models.Student, 0)

	for rows.Next() {
		var student models.Student
		err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			fmt.Printf("Error scanning student row: %v\n", err)
			// http.Error(w, "Failed to scan student", http.StatusInternalServerError)
			return nil, 0,utils.ErrorHandler(err, "Failed to scan student")
		}
		students = append(students, student)
	}

	var totalStudents int
	err = db.QueryRow("SELECT COUNT(*) FROM students").Scan(&totalStudents)
	if err!=nil{
		utils.ErrorHandler(err,"")
		totalStudents = 0;
	}


	return students,totalStudents, nil
}

func GetStudentByID(id int) (models.Student, error) {
	db, err := ConnectReadDb("schooldb") // SELECT — replica is fine
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close
	var student models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
	if err != nil {
		if err == sql.ErrNoRows {
			// http.Error(w, "Student not found", http.StatusNotFound)
			return models.Student{}, utils.ErrorHandler(err, "Student not found")
		}
		// http.Error(w, "Failed to query student", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to query student")
	}
	return student, nil
}




func AddStudentsDBHandler(newStudents []models.Student) ([]models.Student, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	// stmnt, err := db.Prepare("INSERT INTO students (first_name, last_name, email, class) VALUES (?, ?, ?, ?)")
	stmnt, err := db.Prepare(utils.GenerateInsertQuery("students", models.Student{}))

	if err != nil {
		// http.Error(w, "Failed to prepare statement", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Failed to prepare statement")
	}
	defer stmnt.Close()

	addedStudents := make([]models.Student, len(newStudents))
	for i, newStudent := range newStudents {
		// res, err := stmnt.Exec(newStudent.FirstName, newStudent.LastName, newStudent.Email, newStudent.Class)
		values:= utils.GetStructValues(newStudent)
		res, err := stmnt.Exec(values...)

		if err != nil {
			fmt.Println("---- ERROR----",err.Error())
			if strings.Contains(err.Error(), "a foreign key constraint fails (`school`.`students` CONSTRAINT `students_ibk_1` FOREIGN KEY (`class`) REFERENCES `teachers` (`class`))") {
					return nil, utils.ErrorHandler(err, "Invalid class value: no teacher assigned to this class or class does not exist!")
			}
			// http.Error(w, "Failed to insert student", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to insert student")
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			// http.Error(w, "Failed to retrieve last insert id", http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to retrieve last insert id")
		}
		newStudent.ID = int(lastId)
		addedStudents[i] = newStudent
	}
	return addedStudents, nil
}



func UpdateStudent(id int, updatedStudent models.Student) (models.Student, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&existingStudent.ID, &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)
	if err == sql.ErrNoRows {
		// http.Error(w, "Student not found", http.StatusNotFound)
		return models.Student{}, utils.ErrorHandler(err, "Student not found")
	} else if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve student", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to retrieve student")
	}
	updatedStudent.ID = existingStudent.ID
	_, err = db.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Email, updatedStudent.Class, updatedStudent.ID)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to update student")
	}
	return updatedStudent, nil
}

func PatchStudents(updates []map[string]interface{}) error {
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
			// http.Error(w, "Invalid or missing student ID", http.StatusBadRequest)
			return utils.ErrorHandler(err, "Invalid or missing student ID")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			tx.Rollback()
			// http.Error(w, "Invalid student ID", http.StatusBadRequest)
			return utils.ErrorHandler(err, "Invalid student ID")
		}
		var studentFromDb models.Student
		err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", int(id)).Scan(&studentFromDb.ID, &studentFromDb.FirstName, &studentFromDb.LastName, &studentFromDb.Email, &studentFromDb.Class)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {

				// http.Error(w, fmt.Sprintf("Student with ID %d not found", id), http.StatusNotFound)
				return utils.ErrorHandler(err, "Student not found")
			}
			log.Println(err)
			// http.Error(w, "Failed to retrieve student", http.StatusInternalServerError)
			return utils.ErrorHandler(err, "Failed to retrieve student")
		}
		// apply updates to studentFromDb using reflect package

		studentVal := reflect.ValueOf(&studentFromDb).Elem()
		studentType := studentVal.Type()

		for k, v := range update {
			if k == "id" {
				continue
			}
			for i := 0; i < studentVal.NumField(); i++ {
				field := studentType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := studentVal.Field(i)
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
		_, err = tx.Exec("UPDATE students SET first_name=?,last_name=?, email=?, class=? WHERE id=?", studentFromDb.FirstName, studentFromDb.LastName, studentFromDb.Email, studentFromDb.Class, studentFromDb.ID)
		if err != nil {
			tx.Rollback()
			log.Println(err)
			// http.Error(w, "Failed to update student", http.StatusInternalServerError)
			return utils.ErrorHandler(err, "Failed to update student")
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

func PatchOneStudent(id int, updates map[string]interface{}) (models.Student, error) {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&existingStudent.ID, &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)
	if err == sql.ErrNoRows {
		// http.Error(w, "Student not found", http.StatusNotFound)
		return models.Student{}, utils.ErrorHandler(err, "Student not found")
	} else if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve student", http.StatusInternalServerError)
		return models.Student{

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
		}, utils.ErrorHandler(err,"Failed to retrieve student")
	}

	// Apply updates using reflect package

	studentVal := reflect.ValueOf(&existingStudent).Elem()
	studentType := studentVal.Type()

	for k, v := range updates {
		for i := 0; i < studentVal.NumField(); i++ {
			fmt.Println("k from reflect mechanism", k)
			field := studentType.Field(i)
			fmt.Println(field.Tag.Get("json"))
			if field.Tag.Get("json") == k+",omitempty" {
				if studentVal.Field(i).CanSet() {
					studentVal.Field(i).Set(reflect.ValueOf(v).Convert(studentVal.Field(i).Type()))
				}
			}

		}
	}

	_, err = db.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", existingStudent.FirstName, existingStudent.LastName, existingStudent.Email, existingStudent.Class, existingStudent.ID)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return models.Student{}, utils.ErrorHandler(err, "Failed to update student")
	}
	return existingStudent, nil
}

func DeleteOneStudent(id int) error {
	db, err := ConnectDb("schooldb")
	if err != nil {
		// http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to connect to database")
	}
	// shared pool — do not close

	res, err := db.Exec("DELETE FROM students WHERE id = ?", id)
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to delete student", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to delete student")
	}
	fmt.Println(res.RowsAffected())
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		// http.Error(w, "Failed to retrieve affected rows", http.StatusInternalServerError)
		return utils.ErrorHandler(err, "Failed to retrieve affected rows")
	}
	if rowsAffected == 0 {
		// http.Error(w, "Student not found", http.StatusNotFound)
		return utils.ErrorHandler(err, "Student not found")
	}
	return nil
}

func DeleteStudents(ids []int) ([]int, error) {
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
	stmnt, err := tx.Prepare("DELETE FROM students WHERE id = ?")
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
			// http.Error(w, "Failed to delete student with ID "+strconv.Itoa(id), http.StatusInternalServerError)
			return nil, utils.ErrorHandler(err, "Failed to delete student with ID "+strconv.Itoa(id))
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
			// http.Error(w, "Student with ID "+strconv.Itoa(id)+" not found, does not exist!!", http.StatusNotFound)
			return nil, utils.ErrorHandler(err, "Student with ID "+strconv.Itoa(id)+" not found, does not exist!!")
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
		// http.Error(w, "ID's do not exist and No students were deleted", http.StatusBadRequest)
		return nil, utils.ErrorHandler(err, "ID's do not exist and No students were deleted")
	}
	return deletedIds, nil
}

