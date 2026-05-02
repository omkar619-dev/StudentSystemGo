package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"restapi/internal/cache"
	"restapi/internal/models"
	"restapi/internal/repository/sqlconnect"
	"restapi/pkg/utils"
)

// func ExecsHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case http.MethodGet:
// 		w.Write([]byte("Hello GET Method on execs route!"))
// 		fmt.Println("Hello GET Method on execs route!")
// 		return
// 	case http.MethodPost:
// 		w.Write([]byte("Hello POST Method on execs route!"))
// 		fmt.Println("Hello POST Method on execs route!")
// 		return
// 	case http.MethodPut:
// 		w.Write([]byte("Hello PUT Method on execs route!"))
// 		fmt.Println("Hello PUT Method on execs route!")
// 		return
// 	case http.MethodDelete:
// 		w.Write([]byte("Hello DELETE Method on execs route!"))
// 		fmt.Println("Hello DELETE Method on execs route!")
// 		return
// 	case http.MethodPatch:
// 		w.Write([]byte("Hello PATCH Method on execs route!"))
// 		fmt.Println("Hello PATCH Method on execs route!")
// 		return
// 	}
// 	w.Write([]byte("Hello, Execs route!"))
// 	fmt.Println("Hello execs route")

// }

	func GetExecsHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// logic to get all teachers
	// firstName := r.URL.Query().Get("first_name")
	// lastName := r.URL.Query().Get("last_name")
		// ── Cache-aside ───────────────────────────────────────
		cacheKey := cache.KeyPrefix + "execs:list:" + r.URL.RawQuery
		var execs []models.Exec
		err := cache.GetOrFetch(r.Context(), cacheKey, cache.DefaultTTL, &execs, func() (any, error) {
			var fresh []models.Exec
			fresh, err := sqlconnect.GetExecsDBHandler(fresh, r)
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
		Data []models.Exec `json:"data"`
	}{
	Status: "success",
	Count: len(execs),
	Data: execs,
	}
	w.Header().Set("Content-Type", "application/json")
	// encode the response as JSON and write it to the response writer
	json.NewEncoder(w).Encode(respone) 

	fmt.Println("GET /execs called") 
}

func GetOneExecHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// logic to get all execs
	idStr := r.PathValue("id")
	fmt.Printf("Extracted id string: %s\n", idStr)

// handle path parameters
 id,err := strconv.Atoi(idStr)
 if err!= nil{
	fmt.Printf("Error converting id string to int: %v\n", err)
	http.Error(w, "Invalid student ID", http.StatusBadRequest)
	return
 }
exec, err := sqlconnect.GetExecByID(id)
if err != nil {
	fmt.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
w.Header().Set("Content-Type", "application/json")
//  exec, exists := execs[id]
//  if !exists{
// 	http.Error(w,"Exec not found", http.StatusNotFound)
// 	return
//  }
 json.NewEncoder(w).Encode(exec) 
	fmt.Println("GET /execs called") 
}

func AddExecsHandler(w http.ResponseWriter, r *http.Request) {
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


	var newExecs []models.Exec
	var rawExecs []map[string]interface{}

	body,err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()



	err = json.Unmarshal(body, &rawExecs)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fields := GetFieldNames(models.Exec{})
alloweFields := make(map[string]struct{})
for _,field := range fields{
	alloweFields[field] = struct{}{}
}

for _,exec := range rawExecs{
	for key := range exec{
		_,ok := alloweFields[key]
		if !ok {
			http.Error(w, "Invalid field in request body, only use allowed fields", http.StatusBadRequest)
			return
		}
	}
}

	err = json.Unmarshal(body, &newExecs)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _,exec := range newExecs{
		// if exec.FirstName == "" || exec.LastName == "" || exec.Email == "" || exec.Class == "" || exec.Subject == "" {
		// 	http.Error(w, "Missing required fields in request body", http.StatusBadRequest)
		// 	return
		// }
		err := CheckBlankFields(exec)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}


	addedExecs, err := sqlconnect.AddExecsDBHandler(newExecs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"execs:list:*")

	w.Header().Set("Content-type","application/json")
	w.WriteHeader(http.StatusCreated)
	respone := struct{
	Status string `json:"status"`
	Count int `json:"count"` 	
	Data []models.Exec `json:"data"`
}{
	Status: "success",
	Count: len(addedExecs),
	Data: addedExecs,
}
json.NewEncoder(w).Encode(respone)
}

// //PUT /students/{id}
// func UpdateExecHandler(w http.ResponseWriter, r *http.Request) {
//  idStr := r.PathValue("id")
//  id, err := strconv.Atoi(idStr)
//  if err != nil {
// 	log.Println(err)
// 	http.Error(w, "Invalid student ID", http.StatusBadRequest)
// 	return
//  }

//  var updatedStudent models.Student
//  err = json.NewDecoder(r.Body).Decode(&updatedStudent)
//  if err != nil {
// 	 http.Error(w, "Invalid request body", http.StatusBadRequest)
// 	 return
//  }

//  updatedStudentFromDB,err := sqlconnect.UpdateStudent(id, updatedStudent)
//  if err != nil {
// 	log.Println(err)
// 	 http.Error(w, err.Error(), http.StatusInternalServerError)
// 	 return
//  }
// w.Header().Set("Content-Type", "application/json")
// json.NewEncoder(w).Encode(updatedStudentFromDB)

// }

//PATCH /execs/{id}
func PatchOneExecsHandler(w http.ResponseWriter, r *http.Request){
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
	http.Error(w, "Invalid Exec ID", http.StatusBadRequest)
	return
 }

 var updates map[string]interface{}
 err = json.NewDecoder(r.Body).Decode(&updates)
 if err != nil {
	 http.Error(w, "Invalid request body", http.StatusBadRequest)
	 return
 }

 updatedExec, err := sqlconnect.PatchOneExec(id, updates)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }
cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"execs:list:*")
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(updatedExec)
}



func DeleteOneExecHandler(w http.ResponseWriter, r *http.Request){
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
	http.Error(w, "Invalid Exec ID", http.StatusBadRequest)
	return
 }


 err = sqlconnect.DeleteOneExec(id)
 if err != nil {
 	log.Println(err)
 	http.Error(w, err.Error(), http.StatusInternalServerError)
 	return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"execs:list:*")

	w.WriteHeader(http.StatusNoContent)

	w.Header().Set("Content-Type", "application/json")
	response := struct{
	Status string `json:"status"`
	ID int `json:"id"` }{
		Status: "Exec successfully deleted",
		ID: id,
	}
	json.NewEncoder(w).Encode(response)	 
}



func PatchExecsHandler(w http.ResponseWriter, r *http.Request){
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

 err = sqlconnect.PatchExecs(updates)
 if err != nil {
	log.Println(err)
	 http.Error(w, err.Error(), http.StatusInternalServerError)
	 return
 }

	cache.InvalidatePattern(r.Context(), cache.KeyPrefix+"execs:list:*")

	w.WriteHeader(http.StatusNoContent)
}

func LoginHandler(w http.ResponseWriter, r *http.Request){

	var req models.Exec
	// Data validation
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Username == "" || req.Password == "" { 	
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// search for the exec in the database using the provided email
	user, err := sqlconnect.GetUserByUsername(req.Username)
	if err != nil{
		http.Error(w,"Username and password are invalid",http.StatusBadRequest)
		return
	}

	// if user is active
	if user.InactiveStatus {
		http.Error(w, "user is inactive", http.StatusForbidden)
		return
	}

	//verify password

	err = utils.VerifyPassword(req.Password, user.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// generate JWT token

	tokenString, err := utils.SignToken(user.ID, user.Username, user.Role)
	if err != nil {
		utils.ErrorHandler(err, "failed to generate JWT token")
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	if tokenString == "" {
		utils.ErrorHandler(errors.New("failed to generate JWT token"), "failed to generate JWT token")
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}


	// send token as a response or as a cookie
	http.SetCookie(w, &http.Cookie{
		Name: "Bearer",
		Value: tokenString,
		Path: "/",
		HttpOnly: true,
		Secure: true,
		SameSite: http.SameSiteStrictMode,
		Expires: time.Now().Add(time.Hour),
})

http.SetCookie(w, &http.Cookie{
		Name: "test",
		Value: "testing",
		Path: "/",
		HttpOnly: true,
		Secure: true,
		SameSite: http.SameSiteStrictMode,
		Expires: time.Now().Add(time.Hour),
})
 w.Header().Set("Content-Type", "application/json")
	response := struct{
	Token string `json:"token"`
	 }{
		Token: tokenString,
		
	}
	json.NewEncoder(w).Encode(response)
}


func LogOutHandler(w http.ResponseWriter, r *http.Request){
	http.SetCookie(w, &http.Cookie{
		Name: "Bearer",
		Value:"",
		Path: "/",
		HttpOnly: true,
		Secure: true,
		Expires: time.Unix(0,0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type","application/json")
	w.Write([]byte(`{"message: LOgged out successfully"}`))
}

func UpdatePasswordHandler(w http.ResponseWriter,r *http.Request){
	idStr := r.PathValue("id")
	userId,err := strconv.Atoi(idStr)
	if err != nil{
		http.Error(w,"Invalid exec ID", http.StatusBadRequest)
		return
	}

	var req models.UpdatePassword
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil{
		http.Error(w,"Invalid request body",http.StatusBadRequest)
		return
	}
	r.Body.Close()

	if req.CurrentPassword == "" || req.NewPassword ==""{
		http.Error(w,"Please enter a password",http.StatusBadRequest)
		return
	}

	_,err = sqlconnect.UpdatePasswordinDB(userId, req.CurrentPassword,req.NewPassword)
	if err != nil{
		http.Error(w,err.Error(),http.StatusBadRequest)
return
	}


// 		// send token as a response or as a cookie
// 	http.SetCookie(w, &http.Cookie{
// 		Name: "Bearer",
// 		Value: token,
// 		Path: "/",
// 		HttpOnly: true,
// 		Secure: true,
// 		SameSite: http.SameSiteStrictMode,
// 		Expires: time.Now().Add(time.Hour),
// })


 w.Header().Set("Content-Type", "application/json")
	response := struct{
	Message string `json:"message"`
	 }{
		Message: "password updpated successfully",
		
	}
	json.NewEncoder(w).Encode(response)
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request){
 var req struct {
	Email string `json:"email"`
 }
 err := json.NewDecoder(r.Body).Decode(&req)
 if err !=nil{
	http.Error(w,"Invalid request body",http.StatusBadRequest)
	return
 }
 r.Body.Close()

err = sqlconnect.ForgotPasswordDBHandler(req.Email)
if err != nil {
	http.Error(w,err.Error(),http.StatusBadRequest)
	return
}

// resopond with success message
	fmt.Fprintf(w,"Password reset link sent to %s",req.Email)
}
type request struct{
		NewPassword string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request){
	token := r.PathValue("resetcode")
	

	var req request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err !=nil{
		http.Error(w,"Invalid values in request",http.StatusBadRequest)
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		http.Error(w,"Password should match",http.StatusBadRequest)
		return
	}



	err = sqlconnect.ResetPasswordDBHandler(token, req.NewPassword)
	if err!=nil {
		http.Error(w,err.Error(),http.StatusBadRequest)
		return
	}
	fmt.Fprintln(w,"Password reset successfully")

}


