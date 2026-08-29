# go-backend

The Go API for Gradle, built on [Fiber](https://gofiber.io). It's the only service that talks
to Postgres: it owns auth, the REST API the mobile app uses, S3 uploads/presigned URLs, and
pushing jobs onto Redis for `python-backend` to pick up.

See the root [`README.md`](../README.md) for how this fits into the rest of the system, and
[`API_CONTRACT.md`](../API_CONTRACT.md) for the exact internal contract with `python-backend`.

## Requirements

- Go 1.26+ (see `go.mod`)
- Postgres
- Redis
- S3-compatible object storage (MinIO for local dev)

## Configuration

All config is env vars, loaded via `.env` in this directory (see `internal/config/config.go`).

| Variable | Required | Default | Notes |
|---|---|---|---|
| `PORT` | no | `8080` | |
| `DATABASE_URL` | **yes** | — | e.g. `postgres://user:pass@localhost:5432/gradle?sslmode=disable` |
| `JWT_SECRET` | **yes** | — | signs auth tokens |
| `JWT_EXPIRY_HOURS` | no | `24` | |
| `REDIS_URL` | no | `redis://localhost:6379/0` | job queue |
| `INTERNAL_API_TOKEN` | **yes** | — | shared secret the worker sends as `X-Internal-Token` |
| `S3_ENDPOINT_URL` | no | `http://localhost:9000` | used for the backend's own reads/writes |
| `S3_PUBLIC_URL` | no | falls back to `S3_ENDPOINT_URL` | used when *signing* URLs handed to clients — e.g. an Android emulator needs `10.0.2.2:9000` where the backend itself uses `localhost:9000` |
| `S3_ACCESS_KEY` | no | `minioadmin` | |
| `S3_SECRET_KEY` | no | `minioadmin` | |
| `S3_BUCKET` | no | `gradle-artifacts` | |
| `S3_REGION` | no | `us-east-1` | |
| `S3_VIRTUAL_HOST_STYLE` | no | `false` | set `true` for providers that only support virtual-host addressing (`bucket.host.com/key`) instead of path-style (`host.com/bucket/key`) — Railway's bucket storage needs this; local MinIO doesn't |
| `CORS_ORIGINS` | no | `http://localhost:5173` | comma-separated list of origins allowed to call the API from a browser (the `dashboard/` dev server) |

## Running locally

```sh
go run ./cmd/api
```

### Migrations

Schema lives in `migrations/` ([golang-migrate](https://github.com/golang-migrate/migrate)
format — paired `.up.sql`/`.down.sql` files). Install the CLI (`brew install golang-migrate` or
`go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest`), then:

```sh
migrate -path migrations -database "$DATABASE_URL" up
```

Add a new migration with `migrate create -ext sql -dir migrations -seq <name>`.

## API surface

Full request/response shapes are in the handler code (`internal/handlers/`); this is just the
route map (`internal/router/router.go`).

**Public** (`/api/auth/...`) — register, login.

**Authenticated** (JWT bearer token, `/api/...`):
- `GET /api/auth/me`
- `GET /api/dashboard` — weekly activity summary for the logged-in account (see the root
  README's note on the current student/teacher role gap)
- `GET /api/assignments`, `POST /api/assignments`, `GET /api/assignments/:id` — list/create/detail,
  including computed status (`pending` / `expired` / `submitted` / `graded`) derived from due
  date + latest submission; creating requires an existing `subject_id`
- `GET /api/assignments/:id/submissions`, `POST /api/assignments/:id/submissions` — one
  submission ("answer paper") per (assignment, student) pair; posting again for the same
  student resumes their existing submission instead of creating a duplicate. If the
  `student_name` matches a roster student (case-insensitive) for the same teacher, the
  submission is auto-linked to that student's `id`
- `GET /api/submissions/:id`, `POST /api/submissions/:id/pages`, `GET /api/submissions/:id/composited`
- `GET /api/subjects`, `POST /api/subjects` — a teacher's subjects (e.g. "Math"); assignments
  belong to one
- `GET /api/students`, `POST /api/students` — a teacher's student roster
- `POST /api/students/:id/enroll` — enroll a roster student into a subject
- `GET /api/students/:id` — a student's enrolled subjects plus every assignment in those
  subjects with their submission status (including assignments not yet started)

**Internal** (`X-Internal-Token` header, `/internal/...`) — consumed only by `python-backend`;
see `API_CONTRACT.md` for the full flow.

## Deployment

Deployed on Railway; infrastructure is defined in [`../.railway/railway.ts`](../.railway/railway.ts)
and built from this directory's `Dockerfile`. The `Dockerfile` also bundles the `migrate` CLI
alongside the API binary, since Railway's Postgres doesn't expose a public connection string —
migrations get run in-place via `railway ssh --service go-backend -- sh -c './migrate -path ./migrations -database "$DATABASE_URL" up'`.
