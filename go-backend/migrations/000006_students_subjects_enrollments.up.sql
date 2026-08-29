CREATE TABLE subjects (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, name)
);

CREATE TABLE students (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id),
    name text NOT NULL,
    email text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, name)
);

CREATE TABLE enrollments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id uuid NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    subject_id uuid NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (student_id, subject_id)
);

ALTER TABLE assignments ADD COLUMN subject_id uuid REFERENCES subjects(id);

-- Backfill: turn each distinct free-text subject per teacher into a real
-- subject row, then point the assignment at it.
INSERT INTO subjects (owner_id, name)
SELECT DISTINCT owner_id, subject FROM assignments
WHERE subject IS NOT NULL AND subject <> ''
ON CONFLICT (owner_id, name) DO NOTHING;

UPDATE assignments a SET subject_id = s.id
FROM subjects s
WHERE s.owner_id = a.owner_id AND s.name = a.subject;

ALTER TABLE assignments DROP COLUMN subject;

ALTER TABLE submissions ADD COLUMN student_id uuid REFERENCES students(id);
