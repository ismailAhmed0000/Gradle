ALTER TABLE submissions DROP COLUMN student_id;

ALTER TABLE assignments ADD COLUMN subject text;
UPDATE assignments a SET subject = s.name FROM subjects s WHERE s.id = a.subject_id;
ALTER TABLE assignments DROP COLUMN subject_id;

DROP TABLE enrollments;
DROP TABLE students;
DROP TABLE subjects;
