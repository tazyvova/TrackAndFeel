# Local CI/CD with `act`

This project supports running GitHub Actions locally using [`act`](https://github.com/nektos/act). This is pre-installed in the Dev Container.

## Prerequisites

- Docker must be running on your host machine.
- You should be inside the Dev Container.

## Running Workflows

To see all available workflows, run:
```bash
act --list
```

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
