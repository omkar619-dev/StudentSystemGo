-- Migration 1: initial schema.
--
-- Mirrors what was previously in scripts/init.sql, but with IF NOT EXISTS
-- so it's idempotent — running it on an existing populated DB is a no-op.
-- This matters during the transition: EC2 already has these tables; we mark
-- v1 as applied manually (INSERT INTO schema_migrations VALUES (1, 0)) and
-- migrate skips this file. Fresh local volumes will run it and create tables.

-- ── teachers ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS teachers (
  id          INT(11)      NOT NULL AUTO_INCREMENT,
  first_name  VARCHAR(255) NOT NULL,
  last_name   VARCHAR(255) NOT NULL,
  email       VARCHAR(255) NOT NULL,
  class       VARCHAR(255) NOT NULL,
  subject     VARCHAR(255) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY  uq_teachers_email (email),
  KEY         idx_teachers_class (class)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;

-- ── students ────────────────────────────────────────────────
-- FK on class → teachers.class enforces "students can only be in classes
-- that have an assigned teacher." Same as before.
CREATE TABLE IF NOT EXISTS students (
  id          INT(11)      NOT NULL AUTO_INCREMENT,
  first_name  VARCHAR(255) NOT NULL,
  last_name   VARCHAR(255) NOT NULL,
  email       VARCHAR(255) NOT NULL,
  class       VARCHAR(255) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY  uq_students_email (email),
  KEY         idx_students_class (class),
  CONSTRAINT  fk_students_class FOREIGN KEY (class) REFERENCES teachers (class)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;

-- ── execs (admin/manager users with login credentials) ─────
CREATE TABLE IF NOT EXISTS execs (
  id                          INT(11)      NOT NULL AUTO_INCREMENT,
  first_name                  VARCHAR(255) NOT NULL,
  last_name                   VARCHAR(255) NOT NULL,
  email                       VARCHAR(255) NOT NULL,
  username                    VARCHAR(255) NOT NULL,
  password                    VARCHAR(255) NOT NULL,
  password_changed_at         TIMESTAMP    NULL DEFAULT NULL,
  user_created_at             TIMESTAMP    NULL DEFAULT CURRENT_TIMESTAMP(),
  password_reset_code         VARCHAR(255) DEFAULT NULL,
  password_reset_code_expires TIMESTAMP    NULL DEFAULT NULL,
  inactive_status             TINYINT(1)   NOT NULL DEFAULT 0,
  role                        VARCHAR(255) NOT NULL,
  password_reset_token        VARCHAR(255) DEFAULT NULL,
  password_token_expires      DATETIME     DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY  uq_execs_email (email),
  UNIQUE KEY  uq_execs_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
