# Local CI/CD with `act`

This project is configured with a dedicated Dev Container that has [`act`](https://github.com/nektos/act) pre-installed.

## How to use it

1. Open in Dev Container:
   - Press `F1` (or `Ctrl+Shift+P`).
   - Run `Dev Containers: Reopen in Container`.
   - Wait for the build to finish. You are inside when VS Code shows `Dev Container: TrackAndFeel Dev`.
2. Open a terminal (`Ctrl+Shift+\`` or terminal menu) inside the dev container.
3. Confirm services are running:

```bash
docker compose ps
```

The compose stack starts `db`, `backend`, and `frontend` automatically for this setup.

## Running workflows

To see all available workflows:

```bash
act --list
```

### Simulate a pull request

```bash
act pull_request
```

### Simulate a push to `master`

```bash
act push -b master
```

### Run a specific job

```bash
act -j backend
```

## Troubleshooting

- Large runner image: on first run, `act` asks for image size. `Medium` is usually a good balance.
- Docker socket: `act` requires Docker socket access; it is mounted in `.devcontainer/docker-compose.extend.yml`.
- Secrets: pass secrets via `.secrets` file or command line, for example `--secret SECRET_NAME=value`.
