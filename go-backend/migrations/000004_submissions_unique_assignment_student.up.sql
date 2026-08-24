ALTER TABLE submissions
    ADD CONSTRAINT submissions_assignment_id_student_name_key UNIQUE (assignment_id, student_name);
