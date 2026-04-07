# Local CI/CD with `act`

This project is configured with a dedicated **Dev Container** that has [`act`](https://github.com/nektos/act) pre-installed.

## How to use it

1.  **Open in Dev Container**: 
    - Press `F1` (or `Ctrl+Shift+P`).
    - Type **"Dev Containers: Reopen in Container"** and press Enter.
    - Wait for the build to finish. You will know you are inside when the bottom-left corner of VS Code says **"Dev Container: TrackAndFeel Dev"**.

2.  **Open Terminal**: 
    - Once inside, open a new terminal (`Ctrl+` `).
    - You are now in a Linux environment with all tools ready.

## Running Workflows

### Simulate a Pull Request

To run the CI workflow as it would run on a pull request:
```bash
act pull_request
```

### Simulate a Push to Master

To run the CI workflow as it would run on a push to the master branch:
```bash
act push -b master
```

### Run a Specific Job

To run only the backend job:
```bash
act -j backend
```

## Troubleshooting

- **Large Runner Image**: The first time you run `act`, it will ask which image size to use. The "Medium" image is usually a good balance.
- **Docker Socket**: `act` requires access to the Docker socket. The Dev Container is configured to mount it via the `docker-outside-of-docker` feature.
- **Secrets**: If your workflow requires secrets, you can provide them using a `.secrets` file or via command line `--secret SECRET_NAME=value`.
