# Internal API Contract (go-backend <-> python-backend)

This is the interface boundary between the two services. python-backend implements the client
side of everything below; go-backend needs to implement the server side to match.

All endpoints in this doc require header `X-Internal-Token: <INTERNAL_API_TOKEN>` (shared
secret env var, constant-time compare on the Go side). They are in addition to the two
`PATCH` endpoints already specified in the original spec (`/internal/answer-regions/:id` and
`/internal/composited-documents/:id`) — four read/lifecycle endpoints were added so the worker
never needs direct DB access (per the architecture note: Go owns "auth, DB access, job
queuing"; the worker only does OpenCV/PyMuPDF processing).

## Redis queue

Two lists. Payload is the **reference_id** (not the processing_jobs.id) — the answer_region.id
for extract_ink jobs, the composited_document.id for composite_pdf jobs — since that's what
every downstream endpoint is keyed by anyway.

- `jobs:extract_ink` — RPUSH `<answer_region_id>` after the answer_regions row + processing_jobs
  row (job_type=extract_ink, reference_id=answer_region_id) are committed.
- `jobs:composite_pdf` — RPUSH `<composited_document_id>` after the composited_documents row +
  processing_jobs row (job_type=composite_pdf, reference_id=composited_document_id) are
  committed.

Worker does `BLPOP jobs:extract_ink jobs:composite_pdf <timeout>`.

## extract_ink flow

**`PATCH /internal/answer-regions/:id/start`**
No body. Go: `answer_regions.status: pending -> processing`; finds the matching
`processing_jobs` row (`job_type=extract_ink, reference_id=:id, status=queued`) and sets it to
`running`. 409 if the answer_region isn't `pending`.

**`GET /internal/answer-regions/:id/context`**
```json
{
  "crop_x": 120.5, "crop_y": 300.0, "crop_width": 400.0, "crop_height": 150.0,
  "source_page": { "raw_image_path": "submissions/<sub_id>/pages/<page_id>.jpg" }
}
```

**`PATCH /internal/answer-regions/:id`** (final result)
Success: `{"status": "done", "extracted_ink_path": "answer-regions/<id>/ink.png"}`
Failure: `{"status": "failed", "error_message": "..."}`
Go: updates `answer_regions.status` (+`extracted_ink_path` if done) and the matching
`processing_jobs` row (`done`/`failed`, `error_message` if failed) in one transaction.

## composite_pdf flow

**`PATCH /internal/composited-documents/:id/start`**
No body. Go: `composited_documents.status: pending -> generating`; matching `processing_jobs`
row -> `running`.

**`GET /internal/composited-documents/:id/context`**
```json
{
  "submission_id": "uuid",
  "version": 1,
  "assignment_files": [
    { "id": "uuid", "file_path": "assignments/<aid>/<fid>.pdf" }
  ],
  "answers": [
    {
      "answer_region_id": "uuid", "question_id": "uuid", "question_number": 1,
      "has_defined_region": true, "assignment_file_id": "uuid", "page_number": 1,
      "region_x": 72.0, "region_y": 400.0, "region_width": 300.0, "region_height": 120.0,
      "extracted_ink_path": "answer-regions/<id>/ink.png"
    },
    {
      "answer_region_id": "uuid2", "question_id": "uuid2", "question_number": 2,
      "has_defined_region": false, "assignment_file_id": "uuid", "page_number": null,
      "region_x": null, "region_y": null, "region_width": null, "region_height": null,
      "extracted_ink_path": "answer-regions/<id2>/ink.png"
    }
  ]
}
```
`version` is the value already on the `composited_documents` row Go created when finalizing the
submission (worker uses it to build the output storage key, doesn't invent it).
`answers` includes only `status=done` answer_regions for this submission, **at most one per
question** — Go dedups to the most recently created `done` region if a student redid a crop.
Region coordinates use PyMuPDF's native top-left-origin point space (`fitz.Rect` / `Page.rect`
convention) — that's what "PDF coordinates" means throughout.

**`PATCH /internal/composited-documents/:id`** (final result)
Success:
```json
{
  "status": "done",
  "file_path": "composited/<submission_id>/v<version>.pdf",
  "pages": [
    {"page_number": 1, "page_type": "original_with_overlay", "question_id": null},
    {"page_number": 2, "page_type": "appended_blank", "question_id": "uuid2"}
  ]
}
```
Failure: `{"status": "failed", "error_message": "..."}`
Go: updates `composited_documents.status` (+`file_path`), inserts `composited_document_pages`
rows from `pages`, and updates the matching `processing_jobs` row, plus flips the parent
`submissions.status` to `composited`/`failed`.

## Object storage

Single bucket (env `S3_BUCKET`, default `gradle-artifacts`), path-style addressing (required
for MinIO). Worker uploads/downloads directly against S3/MinIO itself (this is explicitly in
its scope per the original spec, unlike DB access). Key conventions the worker uses:
- `answer-regions/<answer_region_id>/ink.png` (PNG, alpha = ink mask, RGB = black)
- `composited/<submission_id>/v<version>.pdf`

Go is expected to use the same bucket for its own uploads:
- `assignments/<assignment_id>/<assignment_file_id>.pdf`
- `submissions/<submission_id>/pages/<page_id>.jpg`

## Env vars the worker needs

`REDIS_URL` (default `redis://localhost:6379/0`), `API_BASE_URL` (default
`http://localhost:8080`), `INTERNAL_API_TOKEN` (required, must match Go's), `S3_ENDPOINT_URL`
(default `http://localhost:9000`), `S3_ACCESS_KEY` / `S3_SECRET_KEY` (default `minioadmin` /
`minioadmin` for local MinIO), `S3_BUCKET`, `S3_REGION` (default `us-east-1`, dummy value for
MinIO), `BLPOP_TIMEOUT_SECONDS` (default `5`).
