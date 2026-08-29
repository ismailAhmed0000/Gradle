# dashboard

Teacher-facing web dashboard for Gradle. React + TypeScript, built with Vite.

- **TanStack Router** (file-based routes in `src/routes/`) for routing
- **TanStack Query** for server state
- **Tailwind CSS v4** for styling
- **openapi-fetch** + **openapi-typescript** for a typed API client — see below

## Running locally

```sh
npm install
cp .env.example .env.local   # point VITE_API_BASE_URL at your go-backend, defaults to localhost:8080
npm run dev
```

go-backend needs `CORS_ORIGINS` to include this dev server's origin (defaults to
`http://localhost:5173` already — see `go-backend/README.md`).

## API client / codegen

There's no OpenAPI spec upstream in go-backend, so `openapi/openapi.yaml` is a hand-written
spec that mirrors the actual Fiber routes in `go-backend/internal/router/router.go` and the
request/response shapes in `go-backend/internal/handlers/`. If those change, update the spec
and regenerate:

```sh
npm run codegen   # openapi-typescript openapi/openapi.yaml -> src/api/schema.d.ts
```

`src/api/client.ts` wraps the generated types in an `openapi-fetch` client that attaches the
JWT from `src/lib/auth.ts` to every request and clears it on a 401. Each feature folder under
`src/features/*/api.ts` has thin TanStack Query hooks built on top of that client.

## Structure

```
src/routes/          TanStack Router file-based routes (pages)
src/features/*/api.ts TanStack Query hooks per API resource
src/api/              generated types (schema.d.ts) + the typed fetch client
src/lib/              auth token storage, route guards
src/components/       shared UI bits
openapi/openapi.yaml  hand-written spec used for codegen
```
