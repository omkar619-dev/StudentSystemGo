package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"restapi/internal/repository/sqlconnect"
	"restapi/internal/storage/photo"
	"restapi/pkg/utils"
)

// PresignTeacherPhotoUploadHandler — see students_photo.go for full design notes.
func PresignTeacherPhotoUploadHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid teacher id", http.StatusBadRequest)
		return
	}

	var body struct {
		Ext string `json:"ext"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	ext := normalizeExt(body.Ext)
	if !isAllowedExt(ext) {
		http.Error(w, "unsupported extension; allowed: .jpg .jpeg .png .webp", http.StatusBadRequest)
		return
	}

	key := photo.BuildKey("teachers", id, ext)
	url, err := photo.PresignUpload(r.Context(), key)
	if err != nil {
		log.Printf("[photo] presign upload teacher %d: %v", id, err)
		http.Error(w, "failed to presign upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]string{
		"upload_url": url,
		"key":        key,
	})
}

func ConfirmTeacherPhotoHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid teacher id", http.StatusBadRequest)
		return
	}

	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	expectedPrefix := fmt.Sprintf("photos/teachers/%d/profile", id)
	if !strings.HasPrefix(body.Key, expectedPrefix) {
		http.Error(w, "key does not match this teacher", http.StatusBadRequest)
		return
	}

	if err := sqlconnect.SetTeacherPhotoKey(id, body.Key); err != nil {
		log.Printf("[photo] confirm teacher %d: %v", id, err)
		http.Error(w, "failed to save photo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetTeacherPhotoHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec", "teacher"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid teacher id", http.StatusBadRequest)
		return
	}

	key, err := sqlconnect.GetTeacherPhotoKey(id)
	if err != nil {
		log.Printf("[photo] get teacher %d: %v", id, err)
		http.Error(w, "failed to fetch photo", http.StatusInternalServerError)
		return
	}
	if key == "" {
		http.Error(w, "no photo for this teacher", http.StatusNotFound)
		return
	}

	url, err := photo.PresignCloudFrontGet(key, 5*time.Minute)
	if err != nil {
		log.Printf("[photo] sign CF for teacher %d: %v", id, err)
		http.Error(w, "failed to generate photo url", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]string{
		"url":        url,
		"expires_in": "300",
	})
}

func DeleteTeacherPhotoHandler(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid teacher id", http.StatusBadRequest)
		return
	}

	key, err := sqlconnect.GetTeacherPhotoKey(id)
	if err != nil {
		log.Printf("[photo] delete: lookup teacher %d: %v", id, err)
		http.Error(w, "failed to fetch photo", http.StatusInternalServerError)
		return
	}
	if key == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := sqlconnect.SetTeacherPhotoKey(id, ""); err != nil {
		log.Printf("[photo] delete: clear teacher %d: %v", id, err)
		http.Error(w, "failed to delete photo", http.StatusInternalServerError)
		return
	}

	if err := photo.Delete(r.Context(), key); err != nil {
		log.Printf("[photo] delete: S3 orphan for key %s: %v", key, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
