-- Reverse: drop the photo_s3_key columns.
-- (Will lose any uploaded photo references — actual S3 objects survive.)
ALTER TABLE students DROP COLUMN photo_s3_key;
ALTER TABLE teachers DROP COLUMN photo_s3_key;
ALTER TABLE execs    DROP COLUMN photo_s3_key;
