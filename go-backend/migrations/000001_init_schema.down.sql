DROP TABLE IF EXISTS processing_jobs;
DROP TABLE IF EXISTS composited_document_pages;
DROP TABLE IF EXISTS composited_documents;
DROP TABLE IF EXISTS answer_regions;
DROP TABLE IF EXISTS submission_pages;
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS assignment_files;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS processing_job_status;
DROP TYPE IF EXISTS job_type;
DROP TYPE IF EXISTS composited_page_type;
DROP TYPE IF EXISTS composited_document_status;
DROP TYPE IF EXISTS answer_region_status;
DROP TYPE IF EXISTS submission_status;
DROP TYPE IF EXISTS user_role;

DROP EXTENSION IF EXISTS pgcrypto;
