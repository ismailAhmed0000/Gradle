ALTER TABLE answer_regions
    DROP CONSTRAINT IF EXISTS answer_regions_submission_id_question_id_key;

CREATE INDEX IF NOT EXISTS idx_answer_regions_submission_question
    ON answer_regions (submission_id, question_id, created_at DESC);
