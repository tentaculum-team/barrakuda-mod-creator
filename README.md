development kit for create extensios for barrakuda

## Commands

- `init` — scaffold a new mod (TUI wizard, or `--name/--type/--out/--force` non-interactive).
- `validate <path>` — validate `specs.json` (+ `config.yaml`, if present).
- `sign` — Ed25519-sign a release zip + `specs.json` with a publisher key.
- `keygen` — generate a publisher's Ed25519 key pair.
- `lock generate <path>` — write `mod.lock`, a content-integrity hash over `specs.json` + `config.yaml` + `icon.svg` + source. Run locally and commit whenever any of those change.
- `lock verify <path>` — recompute the lock and fail if it doesn't match what's committed. This is the CI-facing command — see `docs/manifest-schema.md#modlock`.
