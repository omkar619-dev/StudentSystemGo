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

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	City string `json:"city"`
}

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

func GetTeachersHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec", "teacher"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// ── Cache-aside ───────────────────────────────────────
	// Cache key includes RawQuery so different filter variants don't collide:
	//   cache:teachers:list:                 (no filters)
	//   cache:teachers:list:sortby=name      (with filters)
	// Pattern invalidation `cache:teachers:list:*` wipes all variants on writes.
	cacheKey := cache.KeyPrefix + "teachers:list:" + r.URL.RawQuery

	var teachers []models.Teacher
	err := cache.GetOrFetch(r.Context(), cacheKey, cache.DefaultTTL, &teachers, func() (any, error) {
		var fresh []models.Teacher
		fresh, err := sqlconnect.GetTeachersDBHandler(fresh, r)
		return fresh, err
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// for _, teacher := range teachers {
	// 	if(firstName == "" || teacher.FirstName == firstName) && (lastName == "" || teacher.LastName == lastName){
	// 		teachersList = append(teachersList, teacher)
	// 	}
	// }

	respone := struct{
		Status string `json:"status"`
		Count int `json:"count"`
		Data []models.Teacher `json:"data"`
	}{
	Status: "success",
	Count: len(teachers),
	Data: teachers,
	}
	w.Header().Set("Content-Type", "application/json")
	// encode the response as JSON and write it to the response writer
	json.NewEncoder(w).Encode(respone) 

	fmt.Println("GET /teachers called") 
}

func GetOneTeacherHandler(w http.ResponseWriter, r *http.Request) {
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
	idStr := r.PathValue("id")
	fmt.Printf("Extracted id string: %s\n", idStr)

// handle path parameters
 id,err := strconv.Atoi(idStr)
 if err!= nil{
	fmt.Printf("Error converting id string to int: %v\n", err)
	http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
	return
 }
teacher, err := sqlconnect.GetTeacherByID(id)
if err != nil {
	fmt.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
w.Header().Set("Content-Type", "application/json")
//  teacher, exists := teachers[id]
//  if !exists{
// 	http.Error(w,"Teacher not found", http.StatusNotFound)
// 	return
//  }
 json.NewEncoder(w).Encode(teacher) 
	fmt.Println("GET /teachers called") 
}

func AddTeacherHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// mutex.Lock()
	// defer mutex.Unlock()


	var newTeachers []models.Teacher
	var rawTeachers []map[string]interface{}

	body,err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()



	err = json.Unmarshal(body, &rawTeachers)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fields := GetFieldNames(models.Teacher{})
alloweFields := make(map[string]struct{})
for _,field := range fields{
	alloweFields[field] = struct{}{}
}

for _,teacher := range rawTeachers{
	for key := range teacher{
		_,ok := alloweFields[key]
		if !ok {
			http.Error(w, "Invalid field in request body, only use allowed fields", http.StatusBadRequest)
			return
		}
	}
}

	err = json.Unmarshal(body, &newTeachers)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _,teacher := range newTeachers{
		// if teacher.FirstName == "" || teacher.LastName == "" || teacher.Email == "" || teacher.Class == "" || teacher.Subject == "" {
		// 	http.Error(w, "Missing required fields in request body", http.StatusBadRequest)
		// 	return
		// }
		err := CheckBlankFields(teacher)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}


	addedTeachers, err := sqlconnect.AddTeachersDBHandler(newTeachers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache invalidation: writes occurred → list cache is stale.
	// `teachers:list:*` covers all filter variants (e.g., sortby=name).
	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"teachers:list:*")

	w.Header().Set("Content-type","application/json")
	w.WriteHeader(http.StatusCreated)
	respone := struct{
	Status string `json:"status"`
	Count int `json:"count"` 	
	Data []models.Teacher `json:"data"`
}{
	Status: "success",
	Count: len(addedTeachers),
	Data: addedTeachers,
}
json.NewEncoder(w).Encode(respone)
}

//PUT /teachers/{id}
func UpdateTeacherHandler(w http.ResponseWriter, r *http.Request) {
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
	http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
	return
 }

 var updatedTeacher models.Teacher
 err = json.NewDecoder(r.Body).Decode(&updatedTeacher)
 if err != nil {
	 http.Error(w, "Invalid request body", http.StatusBadRequest)
	 return
 }

 updatedTeacherFromDB,err := sqlconnect.UpdateTeacher(id, updatedTeacher)
 if err != nil {
	log.Println(err)
	 http.Error(w, err.Error(), http.StatusInternalServerError)
	 return
 }
cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"teachers:list:*")
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(updatedTeacherFromDB)

}

//PATCH /teachers/{id}
func PatchOneTeacherHandler(w http.ResponseWriter, r *http.Request){
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
	http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
	return
 }

 var updates map[string]interface{}
 err = json.NewDecoder(r.Body).Decode(&updates)
 if err != nil {
	 http.Error(w, "Invalid request body", http.StatusBadRequest)
	 return
 }

 updatedTeacher, err := sqlconnect.PatchOneTeacher(id, updates)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }
cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"teachers:list:*")
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(updatedTeacher)

}



func DeleteOneTeacherHandler(w http.ResponseWriter, r *http.Request){
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
	http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
	return
 }


 err = sqlconnect.DeleteOneTeacher(id)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"teachers:list:*")

	w.WriteHeader(http.StatusNoContent)

	w.Header().Set("Content-Type", "application/json")
	response := struct{
	Status string `json:"status"`
	ID int `json:"id"` }{
		Status: "Teacher successfully deleted",
		ID: id,
	}
	json.NewEncoder(w).Encode(response)
}



func PatchTeachersHandler(w http.ResponseWriter, r *http.Request){
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

 err = sqlconnect.PatchTeachers(updates)
 if err != nil {
	log.Println(err)
	 http.Error(w, err.Error(), http.StatusInternalServerError)
	 return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"teachers:list:*")

	w.WriteHeader(http.StatusNoContent)
}



func DeleteTeachersHandler(w http.ResponseWriter, r *http.Request){
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

 deletedIds, err := sqlconnect.DeleteTeachers(ids)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"teachers:list:*")

	w.Header().Set("Content-Type", "application/json")
	response := struct{
	Status string `json:"status"`
	DeletedIDs []int `json:"deleted_ids"`}{
		Status: "Teachers successfully deleted",
		DeletedIDs: deletedIds,
	}
	json.NewEncoder(w).Encode(response)
}

func GetStudentsByTeacherId(w http.ResponseWriter, r *http.Request){
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec", "teacher"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	teacherid := r.PathValue("id")
	var students []models.Student
	 students, err := sqlconnect.GetStudentsByTeacherIdromDb(teacherid, w, students)
	 if err != nil {
	 	http.Error(w, err.Error(), http.StatusInternalServerError)
	 	return
	 }
 response := struct{
	Status string `json:"status"`
	Count int `json:"count"`
	Data []models.Student `json:"data"`
}{
	Status: "success",
	Count: len(students),
	Data: students,
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
 }

func GetStudentCountByTeacherId(w http.ResponseWriter, r *http.Request){
	// admin,teacher,exec
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
    if !ok {
        http.Error(w, "missing role in context", http.StatusUnauthorized)
        return
    }
	  _, err := utils.AuthorizeUser(role, "admin", "manager", "exec")
    if err != nil {
        http.Error(w, err.Error(), http.StatusForbidden)
        return
    }
	teacherid := r.PathValue("id")
	 studentCount, err := sqlconnect.GetStudentCountByTeacherIdFromDb(teacherid)
	 if err != nil {
	 	http.Error(w, err.Error(), http.StatusInternalServerError)
	 	return
	 }

 response := struct{
	Status string `json:"status"`
	Count int `json:"count"`
}{
	Status: "success",
	Count: studentCount,
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)	
}


