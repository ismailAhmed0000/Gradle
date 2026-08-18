CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM ('teacher', 'admin');
CREATE TYPE submission_status AS ENUM ('pending', 'processing', 'composited', 'failed');
CREATE TYPE answer_region_status AS ENUM ('pending', 'processing', 'done', 'failed');
CREATE TYPE composited_document_status AS ENUM ('pending', 'generating', 'done', 'failed');
CREATE TYPE composited_page_type AS ENUM ('original_with_overlay', 'appended_blank');
CREATE TYPE job_type AS ENUM ('extract_ink', 'composite_pdf');
CREATE TYPE processing_job_status AS ENUM ('queued', 'running', 'done', 'failed');

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role user_role NOT NULL DEFAULT 'teacher',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE assignments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id),
    title text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE assignment_files (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    file_path text NOT NULL,
    page_count integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE questions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    assignment_file_id uuid NOT NULL REFERENCES assignment_files(id) ON DELETE CASCADE,
    question_number integer NOT NULL,
    has_defined_region boolean NOT NULL DEFAULT false,
    page_number integer,
    region_x double precision,
    region_y double precision,
    region_width double precision,
    region_height double precision,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (assignment_id, question_number),
    CHECK (
        has_defined_region = false
        OR (page_number IS NOT NULL AND region_x IS NOT NULL AND region_y IS NOT NULL
            AND region_width IS NOT NULL AND region_height IS NOT NULL)
    )
);

CREATE TABLE submissions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    student_name text NOT NULL,
    status submission_status NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE submission_pages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id uuid NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    page_number integer NOT NULL,
    raw_image_path text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (submission_id, page_number)
);

CREATE TABLE answer_regions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id uuid NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    source_page_id uuid NOT NULL REFERENCES submission_pages(id) ON DELETE CASCADE,
    status answer_region_status NOT NULL DEFAULT 'pending',
    crop_x double precision NOT NULL,
    crop_y double precision NOT NULL,
    crop_width double precision NOT NULL,
    crop_height double precision NOT NULL,
    extracted_ink_path text,
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_regions_submission_question
    ON answer_regions (submission_id, question_id, created_at DESC);

CREATE TABLE composited_documents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id uuid NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    version integer NOT NULL,
    status composited_document_status NOT NULL DEFAULT 'pending',
    file_path text,
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (submission_id, version)
);

CREATE TABLE composited_document_pages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    composited_document_id uuid NOT NULL REFERENCES composited_documents(id) ON DELETE CASCADE,
    page_number integer NOT NULL,
    page_type composited_page_type NOT NULL,
    question_id uuid REFERENCES questions(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE processing_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type job_type NOT NULL,
    reference_id uuid NOT NULL,
    status processing_job_status NOT NULL DEFAULT 'queued',
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_processing_jobs_lookup
    ON processing_jobs (job_type, reference_id, status);
