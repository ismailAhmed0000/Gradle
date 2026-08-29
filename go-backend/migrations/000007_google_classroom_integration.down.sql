ALTER TABLE submissions DROP COLUMN external_submission_id;
ALTER TABLE submissions DROP COLUMN feedback;
ALTER TABLE submissions DROP COLUMN grade;

DROP INDEX idx_assignments_owner_external_id;
ALTER TABLE assignments DROP COLUMN external_course_id;
ALTER TABLE assignments DROP COLUMN external_id;
ALTER TABLE assignments DROP COLUMN source;

DROP INDEX idx_subjects_owner_external_id;
ALTER TABLE subjects DROP COLUMN external_id;

DROP INDEX idx_students_owner_external_id;
ALTER TABLE students DROP COLUMN external_id;
ALTER TABLE students DROP COLUMN user_id;

DROP TABLE google_oauth_tokens;

ALTER TABLE users DROP COLUMN google_sub;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;

-- Postgres can't drop a single enum value; removing 'student' would require
-- recreating the type. Left as a no-op since down-migrations here are only
-- meant to unblock local dev, not to guarantee perfect symmetry.
