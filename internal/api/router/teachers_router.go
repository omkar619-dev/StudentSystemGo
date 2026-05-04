package router
import (
	"net/http"
	"restapi/internal/api/handlers"
	)

func teachersRouter() *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /teachers", handlers.GetTeachersHandler)
	mux.HandleFunc("POST /teachers", handlers.AddTeacherHandler)
	mux.HandleFunc("PATCH /teachers", handlers.PatchTeachersHandler)
	mux.HandleFunc("DELETE /teachers", handlers.DeleteTeachersHandler)

	mux.HandleFunc("PUT /teachers/{id}", handlers.UpdateTeacherHandler)
	mux.HandleFunc("GET /teachers/{id}", handlers.GetOneTeacherHandler)
	mux.HandleFunc("PATCH /teachers/{id}", handlers.PatchOneTeacherHandler)
	mux.HandleFunc("DELETE /teachers/{id}", handlers.DeleteOneTeacherHandler)

	
	mux.HandleFunc("GET /teachers/{id}/students", handlers.GetStudentsByTeacherId)
	mux.HandleFunc("GET /teachers/{id}/studentcount", handlers.GetStudentCountByTeacherId)

	// Photo endpoints — see docs/photo-flow.md
	mux.HandleFunc("POST /teachers/{id}/photo/presign-upload", handlers.PresignTeacherPhotoUploadHandler)
	mux.HandleFunc("POST /teachers/{id}/photo/confirm", handlers.ConfirmTeacherPhotoHandler)
	mux.HandleFunc("GET /teachers/{id}/photo", handlers.GetTeacherPhotoHandler)
	mux.HandleFunc("DELETE /teachers/{id}/photo", handlers.DeleteTeacherPhotoHandler)
return mux
}