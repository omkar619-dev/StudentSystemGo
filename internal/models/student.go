package models

type Student struct{
ID int `json:"id,omitempty" db:"id,omitempty"`
FirstName string `json:"first_name,omitempty" db:"first_name,omitempty"`
LastName string `json:"last_name,omitempty" db:"last_name,omitempty"`
Email string `json:"email,omitempty" db:"email,omitempty"`
Class string `json:"class,omitempty" db:"class,omitempty"`
// PhotoS3Key is the S3 object key (e.g., "photos/students/42/profile.jpg")
// when the student has a profile photo, or empty when they don't.
// We don't return this in API responses — clients get a signed CloudFront
// URL via GET /students/{id}/photo instead.
PhotoS3Key string `json:"-" db:"photo_s3_key,omitempty"`
}