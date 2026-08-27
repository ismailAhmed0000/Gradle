# python-backend

The processing worker for Gradle. It does the actual image/PDF work — OpenCV ink extraction and
PyMuPDF PDF compositing — that `go-backend` doesn't do itself. It never touches Postgres
directly; it talks to `go-backend` over an internal, token-authenticated HTTP API and to
S3-compatible storage directly for file I/O.

See the root [`README.md`](../README.md) for how this fits into the rest of the system, and
[`API_CONTRACT.md`](../API_CONTRACT.md) for the exact wire contract this worker implements.

## Requirements

- Python 3.14 (matches the checked-in `venv/`; anything reasonably recent should work)
- Redis and `go-backend` reachable (it calls back into go-backend's `/internal/...` endpoints)
- S3-compatible object storage (MinIO for local dev)

## Setup

```sh
cd python-backend
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

## Configuration

Env vars, loaded from a `.env` file if present (see `worker/config.py`).

| Variable | Required | Default |
|---|---|---|
| `REDIS_URL` | no | `redis://localhost:6379/0` |
| `API_BASE_URL` | no | `http://localhost:8080` |
| `INTERNAL_API_TOKEN` | **yes** | — must match go-backend's `INTERNAL_API_TOKEN` |
| `S3_ENDPOINT_URL` | no | `http://localhost:9000` |
| `S3_ACCESS_KEY` | no | `minioadmin` |
| `S3_SECRET_KEY` | no | `minioadmin` |
| `S3_BUCKET` | no | `gradle-artifacts` |
| `S3_REGION` | no | `us-east-1` |
| `BLPOP_TIMEOUT_SECONDS` | no | `5` |

## Running

```sh
python -m worker.main
```

This starts an infinite loop: `BLPOP` on `jobs:extract_ink` and `jobs:composite_pdf`, process
whatever comes off, report the result back to go-backend, repeat. There's no supervisor/restart
logic built in — run it under whatever process manager your environment uses.

## Testing

```sh
pytest
```

## What it does

- **`extract_ink`** (`worker/extraction/`) — given an answer region's crop coordinates and the
  scanned page image, isolates the handwritten ink from the paper background (OpenCV) and
  uploads a PNG with the ink as an alpha-masked layer.
- **`composite_pdf`** (`worker/compositing/`) — given a submission's extracted answer regions,
  builds the final graded PDF: ink gets overlaid onto the original question paper page where a
  region was defined, or appended as a new page when it wasn't (PyMuPDF).

`worker/internal_client/` is the HTTP client for go-backend's `/internal/...` endpoints;
`worker/queue/` and `worker/storage/` wrap Redis and S3 respectively.

## Deployment

Not deployed anywhere yet. It needs to run somewhere with access to the same Redis, S3 bucket,
and `INTERNAL_API_TOKEN` as whichever `go-backend` instance it's paired with — otherwise jobs
queued by that backend will never get processed.

`worker/storage/s3_storage.py` hardcodes `addressing_style: "path"` on its boto3 client. That's
fine for local MinIO, but some managed S3-compatible providers (e.g. Railway's bucket storage)
only support virtual-host-style addressing — go-backend hit this exact issue and needed an
`S3_VIRTUAL_HOST_STYLE` flag (see its README). This worker will need the equivalent fix
(`addressing_style: "virtual"` or `"auto"`) before it can be pointed at a provider like that.
