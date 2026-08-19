DROP INDEX IF EXISTS idx_answer_regions_submission_question;

ALTER TABLE answer_regions
    ADD CONSTRAINT answer_regions_submission_id_question_id_key UNIQUE (submission_id, question_id);
