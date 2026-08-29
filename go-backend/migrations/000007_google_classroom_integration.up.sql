ALTER TYPE user_role ADD VALUE 'student';

-- Google-only accounts (students signing in with Google) have no password.
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ADD COLUMN google_sub text UNIQUE;

CREATE TABLE google_oauth_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_encrypted text NOT NULL,
    google_email text NOT NULL,
    scope text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Links a roster row to the real login-capable account created the first
-- time that student signs in with Google, and to their Classroom userId so
-- roster imports can be matched up without relying on email alone.
ALTER TABLE students ADD COLUMN user_id uuid REFERENCES users(id);
ALTER TABLE students ADD COLUMN external_id text;
CREATE UNIQUE INDEX idx_students_owner_external_id
    ON students (owner_id, external_id) WHERE external_id IS NOT NULL;

-- Maps a Gradle subject to the Classroom course it was imported from.
ALTER TABLE subjects ADD COLUMN external_id text;
CREATE UNIQUE INDEX idx_subjects_owner_external_id
    ON subjects (owner_id, external_id) WHERE external_id IS NOT NULL;

-- 'manual' assignments keep working exactly as they do today; 'classroom'
-- assignments are imported read-only and carry the courseWork id they came
-- from so re-running an import doesn't create duplicates.
ALTER TABLE assignments ADD COLUMN source text NOT NULL DEFAULT 'manual';
ALTER TABLE assignments ADD COLUMN external_id text;
ALTER TABLE assignments ADD COLUMN external_course_id text;
CREATE UNIQUE INDEX idx_assignments_owner_external_id
    ON assignments (owner_id, external_id) WHERE external_id IS NOT NULL;

-- The Classroom studentSubmission id is needed to patch a grade onto it and
-- to turn the student's work in; grade/feedback are Gradle's own record of
-- the mark regardless of whether it's a manual or Classroom assignment.
ALTER TABLE submissions ADD COLUMN grade text;
ALTER TABLE submissions ADD COLUMN feedback text;
ALTER TABLE submissions ADD COLUMN external_submission_id text;
