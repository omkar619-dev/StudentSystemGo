-- Migration 1 DOWN: drop everything created in the up file.
-- Order matters: students has FK → teachers, so drop students first.
-- (Down migrations are rarely run in production — they're mostly for
--  local dev rollback. But we keep them honest.)
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS execs;
DROP TABLE IF EXISTS teachers;
