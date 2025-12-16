# TrackAndFeel — Monorepo Skeleton

This is a starter skeleton for a learning project: Go backend + Vue 3 frontend,
containerised and set up for local dev with Vite proxying `/api` to the Go server.

## Codex branch push guard

To force Codex pushes to use the `codex/*` naming convention, enable the provided Git hook:

```
git config core.hooksPath .githooks
```

With the hook enabled, `git push` will fail unless you are on a branch named `codex/<something>` or on `master`.
