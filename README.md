# Gradle

Gradle is a handwriting-grading platform. Teachers manage assignments from a web dashboard;
students log into the mobile app, scan their own handwritten answer sheet with their phone, and
get back a graded PDF with the extracted ink overlaid on the original questions.

## How it fits together

```
mobile-app (React Native)
   │  REST + JWT
   ▼
go-backend (Go / Fiber)  ── owns Postgres, S3 uploads, auth, job queuing
   │  Redis lists (jobs:extract_ink, jobs:composite_pdf)
   ▼
python-backend (Python worker) ── OpenCV ink extraction, PyMuPDF PDF compositing
```

- **go-backend** is the only service that talks to Postgres. It handles auth, serves the REST
  API the mobile app uses, stores uploaded files in S3-compatible storage, and pushes jobs onto
  Redis lists.
- **python-backend** is a worker loop that `BLPOP`s those Redis lists, does the actual image/PDF
  processing (OpenCV + PyMuPDF), and reports results back to go-backend over an internal,
  token-authenticated HTTP API — it never touches Postgres directly.
- **mobile-app** is the React Native client **students** use: view assignments, scan and submit
  their own answer sheet with the camera, view graded results. Teachers use a separate
  **dashboard** (web) to manage assignments — see the note under Repo layout below.

The exact wire contract between go-backend and python-backend (Redis payloads, internal HTTP
endpoints, S3 key conventions) is in [`API_CONTRACT.md`](API_CONTRACT.md) — read that before
touching either side of that boundary.

Each subproject has its own README with setup/run instructions:

- [`go-backend/README.md`](go-backend/README.md)
- [`python-backend/README.md`](python-backend/README.md)
- [`mobile-app/README.md`](mobile-app/README.md)

## Shared infrastructure

Both backend services depend on the same three things, configured independently via env vars
in each service (see their READMEs for the exact variable names):

- **Postgres** — schema owned entirely by go-backend's `migrations/` (golang-migrate).
- **Redis** — the job queue between go-backend and python-backend.
- **S3-compatible object storage** — question papers, scanned pages, extracted ink, and
  composited PDFs. Local dev uses MinIO; production uses Railway's managed bucket storage
  (virtual-host-style addressing — see the `S3_VIRTUAL_HOST_STYLE` note in the go-backend
  README if pointing at a new provider).

For local development, run Postgres, Redis, and MinIO however you'd normally run local
services (Homebrew, Docker, etc.) and point each service's `.env` at them — there's no
docker-compose in this repo yet.

## Deployment

`go-backend` is deployed on Railway; the infrastructure is defined as code in
[`.railway/railway.ts`](.railway/railway.ts) (Postgres, Redis, and the API service, all in one
project). Apply changes with `railway config plan` / `railway config apply` from the repo root.

`python-backend` (the worker) and `mobile-app` are not deployed anywhere yet — the worker needs
to run somewhere with access to the same Redis/S3/go-backend for the grading pipeline to
actually process jobs; until then, uploads on the deployed backend will queue but never finish.

## Repo layout

```
go-backend/       Go API (Fiber) — auth, Postgres, S3, job queuing
python-backend/    Python worker — ink extraction & PDF compositing
mobile-app/        React Native client (students)
dashboard/         Teacher-facing web dashboard — not built yet (empty directory)
fixtures/          Sample PDFs/images used for local testing and seeding (gitignored)
API_CONTRACT.md    Wire contract between go-backend and python-backend
.railway/          Railway infrastructure-as-code (railway.ts)
```

**Note on current state vs. intended audience**: the backend's user model doesn't distinguish
students from teachers yet — `users.role` is `teacher` or `admin` only, assignments belong to
whichever account created them (`owner_id`), and the mobile app's scan flow accepts an arbitrary
student name per submission rather than being tied to the logged-in user's own identity. So
today, the same kind of account drives both the (unbuilt) dashboard and the mobile app; a real
student role/identity model is still needed before the two are actually separated the way the
product intends.
