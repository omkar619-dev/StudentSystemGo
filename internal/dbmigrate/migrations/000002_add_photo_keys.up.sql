-- Migration 2: add photo_s3_key column to students, teachers, execs.
--
-- Stores the S3 object key for the entity's profile photo, e.g.
--   "photos/students/42/profile.jpg"
-- App generates presigned upload URLs against this key; CloudFront serves
-- reads via signed URLs.
--
-- NULL means "no photo uploaded yet." Default NULL is fine.
-- 512 chars is generous — typical S3 keys we'll generate are <100.
ALTER TABLE students ADD COLUMN photo_s3_key VARCHAR(512) NULL;
ALTER TABLE teachers ADD COLUMN photo_s3_key VARCHAR(512) NULL;
ALTER TABLE execs    ADD COLUMN photo_s3_key VARCHAR(512) NULL;
