package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"restapi/internal/repository/sqlconnect"
	"restapi/internal/storage/photo"
	"restapi/pkg/utils"
)

// PresignStudentPhotoUploadHandler issues a presigned S3 PUT URL the client
// can use to upload a profile photo for the given student ID.
//
// Flow from the client's perspective:
//  1. Client: POST /students/42/photo/presign-upload   { "ext": ".jpg" }
//  2. Server replies: { "upload_url": "https://...s3...", "key": "photos/students/42/profile.jpg" }
//  3. Client: PUT <upload_url>  with the image bytes (Content-Type: image/jpeg)
//  4. Client: POST /students/42/photo/confirm  { "key": "..." }   ← stores in DB
//
// We split presign + confirm because:
//   - presign is cheap (no DB write)
//   - if client never actually uploads, we don't pollute DB with stale keys
//   - confirm is the "I successfully PUT — please record this" step
//
// RBAC: admin, manager, or exec only. Students don't log in (no role of their own).
func PresignStudentPhotoUploadHandler(w http.ResponseWriter, r *http.Request) {
	// ── 1. Auth — pull role from JWT context ──────────────
	role, ok := r.Context().Value(utils.ContextKey("role")).(string)
	if !ok {
		http.Error(w, "missing role in context", http.StatusUnauthorized)
		return
	}
	if _, err := utils.AuthorizeUser(role, "admin", "manager", "exec"); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// ── 2. Parse + validate the path param ────────────────
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid student id", http.StatusBadRequest)
		return
	}

	// ── 3. Parse the body — client tells us the file extension ────
	// Why have client send extension? So the S3 key reflects the actual
	// file type (.jpg vs .png). Affects Content-Type browsers infer when
	// they later fetch via CloudFront.
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

	// ── 4. Build the S3 key + presign ─────────────────────
	key := photo.BuildKey("students", id, ext)
	url, err := photo.PresignUpload(r.Context(), key)
	if err != nil {
		log.Printf("[photo] presign upload student %d: %v", id, err)
		http.Error(w, "failed to presign upload", http.StatusInternalServerError)
		return
	}

	// ── 5. Respond ────────────────────────────────────────
	// Echo the key back so the client can pass it verbatim to /confirm
	// (avoids the client trying to reconstruct it).
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // keep & literal in URL (Go default mangles to &)
	_ = enc.Encode(map[string]string{
		"upload_url": url,
		"key":        key,
	})
}

// normalizeExt returns the extension lowercased, with a leading dot,
// or empty string if input was empty.
//   ""       → ""
//   "jpg"    → ".jpg"
//   ".JPG"   → ".jpg"
//   "image.png" → ".png"
func normalizeExt(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// filepath.Ext handles "image.png" → ".png"; if input was already
	// ".jpg" or "jpg", we add the dot back as needed.
	e := filepath.Ext(s)
	if e == "" && !strings.HasPrefix(s, ".") {
		e = "." + s
	} else if e == "" {
		e = s
	}
	return strings.ToLower(e)
}

// isAllowedExt restricts the file types a client can upload.
// Defense in depth — even though S3 doesn't care about extensions, we
// reject obviously-wrong inputs early so users get a clear 400 instead
// of an opaque later failure.
func isAllowedExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	}
	return false
}

// ConfirmStudentPhotoHandler records the S3 key in the DB after the client
// has successfully PUT to the presigned URL.
//
// Flow:
//  1. (earlier) Client got upload_url + key from /presign-upload
//  2. (earlier) Client PUT image bytes to upload_url → S3 stored them
//  3. NOW: Client tells us "I uploaded successfully — please save this key"
//  4. We validate the key matches the expected format for THIS student
//  5. We UPDATE students SET photo_s3_key = ?
//
// RBAC: same as presign — admin/manager/exec.
func ConfirmStudentPhotoHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "invalid student id", http.StatusBadRequest)
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

	// ── Validate the key matches what we'd have issued for this student ──
	// CRITICAL: client controls the body. If we trusted them blindly, an
	// attacker could pass key="photos/execs/1/profile.jpg" while hitting
	// /students/42/photo/confirm — and we'd write that exec's key into
	// student 42's row. Then GET /students/42/photo would serve the exec's
	// photo. Privilege escalation via parameter tampering.
	//
	// Defense: only accept keys that match the expected prefix for THIS
	// entity + ID.
	expectedPrefix := fmt.Sprintf("photos/students/%d/profile", id)
	if !strings.HasPrefix(body.Key, expectedPrefix) {
		http.Error(w, "key does not match this student", http.StatusBadRequest)
		return
	}

	if err := sqlconnect.SetStudentPhotoKey(id, body.Key); err != nil {
		log.Printf("[photo] confirm student %d: %v", id, err)
		http.Error(w, "failed to save photo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 — success, no body
}

// GetStudentPhotoHandler returns a short-lived CloudFront signed URL the
// client can fetch the photo from.
//
// We DON'T just return the S3 key — clients shouldn't know our internal
// structure, and the S3 key isn't directly fetchable anyway (bucket private).
//
// The signed URL is good for 5 minutes. If the user keeps the page open
// longer and refreshes, frontend re-fetches the URL (cheap — local crypto
// only, no DB write).
//
// RBAC: admin/manager/exec/teacher. Teachers can see students' photos for
// classroom purposes. Adjust per your policy.
func GetStudentPhotoHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "invalid student id", http.StatusBadRequest)
		return
	}

	key, err := sqlconnect.GetStudentPhotoKey(id)
	if err != nil {
		log.Printf("[photo] get student %d: %v", id, err)
		http.Error(w, "failed to fetch photo", http.StatusInternalServerError)
		return
	}
	if key == "" {
		http.Error(w, "no photo for this student", http.StatusNotFound)
		return
	}

	url, err := photo.PresignCloudFrontGet(key, 5*time.Minute)
	if err != nil {
		log.Printf("[photo] sign CF for student %d: %v", id, err)
		http.Error(w, "failed to generate photo url", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // keep & literal in URL (Go default mangles to &)
	_ = enc.Encode(map[string]string{
		"url":        url,
		"expires_in": "300", // seconds; client uses this to schedule re-fetch
	})
}

// DeleteStudentPhotoHandler removes the photo for a student.
//
// Order of operations matters. We do DB-first, S3-second.
//
// Why DB first?
//   - If DB succeeds, S3 fails  → DB says "no photo", S3 has orphan object.
//                                 User-visible state: gone. Storage waste:
//                                 small, cleanable via S3 lifecycle policy.
//   - If S3 first, DB fails    → DB still points to deleted key. User
//                                 tries GET → CF tries S3 → S3 returns
//                                 404 → user sees broken image. WORSE.
//
// Rule: prefer "the user-facing state is correct" > "storage is perfectly
// clean." Eventual cleanup is fine; broken UI is not.
//
// RBAC: admin/manager/exec.
func DeleteStudentPhotoHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "invalid student id", http.StatusBadRequest)
		return
	}

	// Read the current key so we know what to delete from S3 afterwards.
	key, err := sqlconnect.GetStudentPhotoKey(id)
	if err != nil {
		log.Printf("[photo] delete: lookup student %d: %v", id, err)
		http.Error(w, "failed to fetch photo", http.StatusInternalServerError)
		return
	}
	if key == "" {
		// Idempotent: no photo to delete is a successful "no-op delete."
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Step 1: clear the DB row first.
	if err := sqlconnect.SetStudentPhotoKey(id, ""); err != nil {
		log.Printf("[photo] delete: clear student %d: %v", id, err)
		http.Error(w, "failed to delete photo", http.StatusInternalServerError)
		return
	}

	// Step 2: best-effort S3 cleanup. If this fails, we LOG but return 204.
	// The user's perspective is correct (photo gone). Orphan can be reaped
	// later by a bucket lifecycle rule, or a periodic cleanup job.
	if err := photo.Delete(r.Context(), key); err != nil {
		log.Printf("[photo] delete: S3 orphan for key %s: %v", key, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
