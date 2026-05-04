package router

import ("net/http"
"restapi/internal/api/handlers")

func studentsRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /students", handlers.GetStudentsHandler)
	mux.HandleFunc("POST /students", handlers.AddStudentHandler)
	mux.HandleFunc("PATCH /students", handlers.PatchStudentsHandler)
	mux.HandleFunc("DELETE /students", handlers.DeleteStudentsHandler)

	mux.HandleFunc("PUT /students/{id}", handlers.UpdateStudentHandler)
	mux.HandleFunc("GET /students/{id}", handlers.GetOneStudentHandler)
	mux.HandleFunc("PATCH /students/{id}", handlers.PatchOneStudentHandler)
	mux.HandleFunc("DELETE /students/{id}", handlers.DeleteOneStudentHandler)

	// Photo endpoints — see docs/photo-flow.md for the full sequence.
	// presign-upload: returns a presigned S3 PUT URL (15-min TTL).
	// confirm:        records the uploaded key in the DB.
	// GET photo:      returns a CloudFront signed URL (5-min TTL).
	// DELETE photo:   clears DB key, deletes S3 object (best-effort).
	mux.HandleFunc("POST /students/{id}/photo/presign-upload", handlers.PresignStudentPhotoUploadHandler)
	mux.HandleFunc("POST /students/{id}/photo/confirm", handlers.ConfirmStudentPhotoHandler)
	mux.HandleFunc("GET /students/{id}/photo", handlers.GetStudentPhotoHandler)
	mux.HandleFunc("DELETE /students/{id}/photo", handlers.DeleteStudentPhotoHandler)
	return mux
}