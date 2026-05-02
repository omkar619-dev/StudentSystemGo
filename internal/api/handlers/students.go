package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"restapi/internal/cache"
	"restapi/internal/models"
	"restapi/internal/repository/sqlconnect"
	"restapi/pkg/utils"
)

// type User struct {
// 	Name string `json:"name"`
// 	Age  int    `json:"age"`
// 	City string `json:"city"`
// }

// var (teachers = make(map[int]models.Teacher)
// mutex = &sync.Mutex{}
// nextID = 1 )

// initialise some dummy data
// func init(){
// 	teachers[nextID] = models.Teacher{ID: nextID, FirstName: "John", LastName: "Doe", Subject: "Math", Class: "10A"}
// 	nextID++
// 	teachers[nextID] = models.Teacher{ID: nextID, FirstName: "Jane", LastName: "Smith", Subject: "Science", Class: "10B"}
// 	nextID++
// 	teachers[nextID] = models.Teacher{ID: nextID, FirstName: "Jane", LastName: "Doe", Subject: "English", Class: "10C"}
// 	nextID++
// }

// func TeachersHandler(w http.ResponseWriter, r *http.Request) {
// 		fmt.Printf("Request method: %s\n", r.Method)
// 		switch r.Method {
// 		case http.MethodGet:
// 			// call get method handler function
// 				getTeachersHandler(w,r)
// 			// w.Write([]byte("Hello GET Method on teachers route!"))
// 			// fmt.Println("Hello GET Method on teachers route!")
// 			return
// 		case http.MethodPost:
// 			addTeacherHandler(w,r)
// 			//POST Request handler
// 			// w.Write([]byte("Hello POST Method on teachers route!"))
// 			// fmt.Println("Hello POST Method on teachers route!")
// 			return
// 		case http.MethodPut:
// 			w.Write([]byte("Hello PUT Method on teachers route!"))
// 			fmt.Println("Hello PUT Method on teachers route!")
// 			updateTeacherHandler(w, r)
// 			return
// 		case http.MethodDelete:
// 			w.Write([]byte("Hello DELETE Method on teachers route!"))
// 			fmt.Println("Hello DELETE Method on teachers route!")
// 			deleteTeacherHandler(w, r)
// 			return
// 		case http.MethodPatch:
// 			w.Write([]byte("Hello PATCH Method on teachers route!"))
// 			fmt.Println("Hello PATCH Method on teachers route!")
// 			patchTeacherHandler(w, r)
// 			return
// 		}
// 		w.Write([]byte("Hello, Teachers route!"))
// 		fmt.Println("Hello teachers route")

// 	}

func GetStudentsHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec", "teacher"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// logic to get all teachers
	// firstName := r.URL.Query().Get("first_name")
	// lastName := r.URL.Query().Get("last_name")

page,limit := getPaginationParams(r)

	// ── Cache-aside ───────────────────────────────────────
	// GetStudentsDBHandler returns 3 values (list, total, err). We wrap
	// list+total in a struct so the cache stores them together — otherwise
	// hits would be missing the total count.
	cacheKey := cache.KeyPrefix + "students:list:" + r.URL.RawQuery
	type cachedStudents struct {
		List  []models.Student `json:"list"`
		Total int              `json:"total"`
	}
	var cached cachedStudents
	err := cache.GetOrFetch(r.Context(), cacheKey, cache.DefaultTTL, &cached, func() (any, error) {
		var fresh []models.Student
		fresh, total, err := sqlconnect.GetStudentsDBHandler(fresh, r, limit, page)
		return cachedStudents{List: fresh, Total: total}, err
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	students := cached.List
	totalStudents := cached.Total

	// for _, teacher := range teachers {
	// 	if(firstName == "" || teacher.FirstName == firstName) && (lastName == "" || teacher.LastName == lastName){
	// 		teachersList = append(teachersList, teacher)
	// 	}
	// }
		// url?limit=50&page=1



	respone := struct{
		Status string `json:"status"`
		Count int `json:"count"`
		Page int  `json:"page"`
		PageSize int  `json:"page_size"`
		Data []models.Student `json:"data"`
	}{
	Status: "success",
	Count: totalStudents,
	Page: page,
	PageSize: limit,
	Data: students,
	}
	w.Header().Set("Content-Type", "application/json")
	// encode the response as JSON and write it to the response writer
	json.NewEncoder(w).Encode(respone) 

	fmt.Println("GET /students called") 
}

func getPaginationParams(r *http.Request)(int,int){
	page,err := strconv.Atoi(r.URL.Query().Get("page"))
	if err!=nil{
		page =1
	}
	limit,err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err!=nil{
		limit =10
	}
	return page,limit
}

func GetOneStudentHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec", "teacher"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// logic to get all students
	idStr := r.PathValue("id")
	fmt.Printf("Extracted id string: %s\n", idStr)

// handle path parameters
 id,err := strconv.Atoi(idStr)
 if err!= nil{
	fmt.Printf("Error converting id string to int: %v\n", err)
	http.Error(w, "Invalid student ID", http.StatusBadRequest)
	return
 }
student, err := sqlconnect.GetStudentByID(id)
if err != nil {
	fmt.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
w.Header().Set("Content-Type", "application/json")
//  student, exists := students[id]
//  if !exists{
// 	http.Error(w,"Student not found", http.StatusNotFound)
// 	return
//  }
 json.NewEncoder(w).Encode(student) 
	fmt.Println("GET /students called") 
}

func AddStudentHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// mutex.Lock()
	// defer mutex.Unlock()


	var newStudents []models.Student
	var rawStudents []map[string]interface{}

	body,err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()



	err = json.Unmarshal(body, &rawStudents)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fields := GetFieldNames(models.Student{})
alloweFields := make(map[string]struct{})
for _,field := range fields{
	alloweFields[field] = struct{}{}
}

for _,student := range rawStudents{
	for key := range student{
		_,ok := alloweFields[key]
		if !ok {
			http.Error(w, "Invalid field in request body, only use allowed fields", http.StatusBadRequest)
			return
		}
	}
}

	err = json.Unmarshal(body, &newStudents)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _,student := range newStudents{
		// if student.FirstName == "" || student.LastName == "" || student.Email == "" || student.Class == "" || student.Subject == "" {
		// 	http.Error(w, "Missing required fields in request body", http.StatusBadRequest)
		// 	return
		// }
		err := CheckBlankFields(student)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}


	addedStudents, err := sqlconnect.AddStudentsDBHandler(newStudents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"students:list:*")

	w.Header().Set("Content-type","application/json")
	w.WriteHeader(http.StatusCreated)
	respone := struct{
	Status string `json:"status"`
	Count int `json:"count"` 	
	Data []models.Student `json:"data"`
}{
	Status: "success",
	Count: len(addedStudents),
	Data: addedStudents,
}
json.NewEncoder(w).Encode(respone)
}

//PUT /students/{id}
func UpdateStudentHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
 idStr := r.PathValue("id")
 id, err := strconv.Atoi(idStr)
 if err != nil {
	log.Println(err)
	http.Error(w, "Invalid student ID", http.StatusBadRequest)
	return
 }

 var updatedStudent models.Student
 err = json.NewDecoder(r.Body).Decode(&updatedStudent)
 if err != nil {
	 http.Error(w, "Invalid request body", http.StatusBadRequest)
	 return
 }

 updatedStudentFromDB,err := sqlconnect.UpdateStudent(id, updatedStudent)
 if err != nil {
	log.Println(err)
	 http.Error(w, err.Error(), http.StatusInternalServerError)
	 return
 }
cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"students:list:*")
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(updatedStudentFromDB)

}

//PATCH /students/{id}
func PatchOneStudentHandler(w http.ResponseWriter, r *http.Request){
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

 idStr := r.PathValue("id")
 id, err := strconv.Atoi(idStr)
 if err != nil {
	log.Println(err)
	http.Error(w, "Invalid student ID", http.StatusBadRequest)
	return
 }

 var updates map[string]interface{}
 err = json.NewDecoder(r.Body).Decode(&updates)
 if err != nil {
	 http.Error(w, "Invalid request body", http.StatusBadRequest)
	 return
 }

 updatedStudent, err := sqlconnect.PatchOneStudent(id, updates)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }
cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"students:list:*")
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(updatedStudent)

}



func DeleteOneStudentHandler(w http.ResponseWriter, r *http.Request){
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

  idStr := r.PathValue("id")
 id, err := strconv.Atoi(idStr)
 if err != nil {
	log.Println(err)
	http.Error(w, "Invalid student ID", http.StatusBadRequest)
	return
 }


 err = sqlconnect.DeleteOneStudent(id)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"students:list:*")

	w.WriteHeader(http.StatusNoContent)

	w.Header().Set("Content-Type", "application/json")
	response := struct{
	Status string `json:"status"`
	ID int `json:"id"` }{
		Status: "Student successfully deleted",
		ID: id,
	}
	json.NewEncoder(w).Encode(response)
}



func PatchStudentsHandler(w http.ResponseWriter, r *http.Request){
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

 var updates []map[string]interface{}

 err := json.NewDecoder(r.Body).Decode(&updates)
 if err != nil {
	 http.Error(w, "Invalid request body", http.StatusBadRequest)
	 return
 }

 err = sqlconnect.PatchStudents(updates)
 if err != nil {
	log.Println(err)
	 http.Error(w, err.Error(), http.StatusInternalServerError)
	 return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"students:list:*")

	w.WriteHeader(http.StatusNoContent)
}



func DeleteStudentsHandler(w http.ResponseWriter, r *http.Request){
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

 var ids []int
 err := json.NewDecoder(r.Body).Decode(&ids)
 if err != nil {
	log.Println(err)
	http.Error(w, "Invalid request body", http.StatusBadRequest)
	return
 }

 deletedIds, err := sqlconnect.DeleteStudents(ids)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"students:list:*")

	w.Header().Set("Content-Type", "application/json")
	response := struct{
	Status string `json:"status"`
	DeletedIDs []int `json:"deleted_ids"`}{
		Status: "Students successfully deleted",
		DeletedIDs: deletedIds,
	}
	json.NewEncoder(w).Encode(response)
}

