# TrackAndFeel — Monorepo Skeleton

This is a starter skeleton for a learning project: Go backend + Vue 3 frontend,
containerised and set up for local dev with Vite proxying `/api` to the Go server.

## Codex branch push guard

To force Codex pushes to use the `codex/*` naming convention, enable the provided Git hook:

```
git config core.hooksPath .githooks
```

With the hook enabled, `git push` will fail unless you are on a branch named `codex/<something>` or on `master`.

## Tests

Run backend tests in a disposable container so nothing lingers after the run:

```
docker compose -f docker-compose.yml -f docker-compose.test.yml run --rm backend-test
```

## Preview environments

Pull requests automatically get an ephemeral preview stack using the `PR Preview Environments` workflow. Key behaviours:

- Backend and frontend images are built from the PR head SHA, tagged as `pr-<number>`, and pushed to GHCR.
- The workflow connects to the VDS over SSH, launches a per-PR Compose project (`trackandfeel-pr-<number>`) using `docker-compose.preview.yml`, and reuses the shared `trackandfeel-preview` ingress network so Caddy can discover the services.
- Caddy uses the wildcard block in `Caddyfile` to proxy `pr-<number>.<preview-domain>` to the matching preview backend/frontend while keeping the `/api`, `/healthz`, `/version`, and SPA behaviour consistent with production.
- A PR comment is posted with the preview URL; the stack and images are removed when the PR closes, and the scheduled GC job prunes stacks older than the configured TTL.

### Required secrets and environment

Set these repository secrets to enable the workflow:

- `VDS_HOST`, `VDS_USER`, `VDS_SSH_KEY`, `VDS_SSH_PASSPHRASE`, `VDS_WORKDIR` – reused from the existing deploy workflows.
- `DB_NAME`, `DB_USER`, `DB_PASSWORD` – credentials for the preview Postgres containers.
- `PREVIEW_DOMAIN` – the wildcard domain (e.g. `preview.example.com`) pointing DNS `*.PREVIEW_DOMAIN` to the product server. Provide an explicit value; use `preview.local` only when testing locally.

Add these values to the server’s `.env` (or export before running compose):

- `PREVIEW_DOMAIN` – shared by Caddy to match wildcard hosts.
- `PREVIEW_INGRESS_NETWORK` (default `trackandfeel-preview`) – the external Docker network that connects the central Caddy container to each preview stack.

Local validation hints:

- `docker compose -f docker-compose.preview.yml --env-file .env.local config` verifies env interpolation for a given PR number.
- Run `./.github/workflows/pr-preview.yml` with `workflow_dispatch` to smoke-test registry pushes and remote deployment without opening a real PR.
