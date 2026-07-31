# Mod manifest schema

A mod is 2 files at the root of its repo, at the tag of a GitHub release:

- **`specs.json`** — identity, capabilities, dependencies. See
  `internal/manifest/manifest.go` (`Manifest` struct) for the authoritative
  field-by-field shape — every field there is `omitempty` except `name`,
  `version`, `type`, so a manifest only needs what its `type` requires (see
  `Manifest.Validate()`).
- **`config.yaml`** (optional) — user-editable settings, see below.

`specs.json` was called `manifest.json` before this split. Ingestion and
`mod-creator validate`/`sign` both still accept `manifest.json` as a
fallback for mods that haven't migrated — there is no forced cutover.

## `.barrakuda/` layout

`specs.json`/`config.yaml` and the two optional files below live under a
`.barrakuda/` folder at the repo root:

```
.barrakuda/
  specs.json      # was: ./specs.json (was: ./manifest.json)
  config.yaml      # was: ./config.yaml
  icon.svg          # optional — source of the `icon` field, see below
  imgs/             # optional — screenshots declared via `banners`
    1.png
    2.png
```

`mod-creator validate`/`sign` and the server's self-service ingestion both
fall back through `.barrakuda/specs.json` → `specs.json` (root) →
`manifest.json` (root, pre-split legacy name) — no forced cutover, older
mods keep working as-is. `config.yaml` falls back through
`.barrakuda/config.yaml` → `config.yaml` (root).

### `icon.svg`

If `.barrakuda/icon.svg` exists, its raw text content becomes the
manifest's `icon` field at ingestion time — same inline-SVG-fragment
format as always (no `<svg>` wrapper, `viewBox="0 0 24 24"`, uses
`currentColor`, sanitized client-side before rendering). Keeping it a real
`.svg` file instead of a JSON string lets an editor/linter actually
understand it while you work on it. If the file doesn't exist, `icon` in
`specs.json` (if set) is used as-is — a fallback, not an error.

### `imgs/` + `banners`

`banners` (`[]string`, root-level, any `type`) declares up to **6**
screenshots shown on the extension's detail page. Each entry is a path
relative to `.barrakuda/` — e.g. `"imgs/1.png"` resolves to
`.barrakuda/imgs/1.png` in this repo. At ingestion, the server resolves
each entry to `https://raw.githubusercontent.com/{owner}/{repo}/{tag}/.barrakuda/{path}`,
fetches it once to confirm it exists and isn't absurdly large, and keeps
only the resolved URL in the published manifest — no proxy, no
server-side image cache; if the source repo goes private later the image
breaks (same trade-off `package.<os>.url` already accepts). Entries
beyond 6, or that fail to resolve, are silently skipped (logged
server-side, never a fatal ingestion error).

```json
{
  "banners": ["imgs/1.png", "imgs/2.png"]
}
```

### `default_color`

`default_color` (string, root-level, any `type`, e.g. `"#ffffff"`) sets
the extension icon's background color in Barrakuda's store. Empty/absent
falls back to the server's fixed neutral gray — self-service ingestion
never prompts for a color, so this field is the only way to pick your
own.

## `config.yaml`

Flat key/value pairs, nothing else — no nested maps, no lists, no
metadata (no label/description/enum/secret/required):

```yaml
api_key: ""
max_tokens: 4000
modo: rapido
```

The widget Barrakuda renders for each field is inferred straight from the
YAML value's own type: `true`/`false` → checkbox, a number → number input,
anything else → text input. There is deliberately no way to declare a
field as a secret (password-masked) or as an enum (dropdown) — this is a
conscious simplicity trade-off: `config.yaml` is meant to be hand-written
in a minute, not modeled as a form schema.

A mod with no `config.yaml` gets no config icon in Barrakuda's mod
library — the file's presence, not its content, is what turns the icon
on.

`config.yaml` is never covered by `mod-creator sign`'s signature (only
`specs.json` is) — it's mutable by design, the client can overwrite it
locally, so signing it would prove nothing.

## `mod.lock`

`mod-creator lock generate <path>` writes a content-integrity lock next to
`specs.json` (`.barrakuda/mod.lock`, or a root `mod.lock` for repos still on
the legacy layout): a sha256 hash covering `specs.json`, `config.yaml` (if
present), `icon.svg` (if present), and every source file in the repo —
changing any of them changes the hash. `mod-creator lock verify <path>`
recomputes it and fails (exit 1) if it no longer matches what's committed —
this is the CI-facing command, wired up in each mod repo's
`.github/workflows/mod-lock.yml`.

Run `lock generate` locally and commit the result whenever `specs.json`,
`config.yaml`, `icon.svg`, or any source file changes. `config.yaml` is
covered here even though it's excluded from `mod-creator sign`'s signature —
the lock's job is drift detection, not release provenance, so a config edit
should trip it.

`icon.svg`'s fallback path mirrors `specs.json`/`config.yaml`:
`.barrakuda/icon.svg` → `icon.svg` (root, legacy layout).

An optional `lockignore` file (`.barrakuda/lockignore`, or a root
`lockignore` for the legacy layout — same fallback order) excludes paths
from the source hash, one glob per line (matched against the file's
basename or its path relative to the repo root, `#`-prefixed lines are
comments). Use it for anything that isn't really source — e.g. compiled
binaries committed for convenience:

```
# lockignore
barrakuda-mod-claude-cli
barrakuda-mod-claude-cli-linux
```

### Reserved key: `allow_machine_access` (agent mods only)

`type: agent` mods with an `agent_provider` block (chat-selectable AI
providers, e.g. Hermes/OpenClaw) run their own reasoning through a local
relay that Barrakuda controls — by default that relay only lets the model
talk, it never lets the mod run real tools (shell/fs/docker) on the user's
machine.

To let a user opt an agent into that (off by default, always denied until
explicitly turned on), declare this boolean in `config.yaml`:

```yaml
allow_machine_access: false
```

There's no dedicated switch for this anymore — it's just another entry in
the same generic config dialog the gear icon opens. If your agent mod
never declares this key, the user has no way to grant it (equivalent to
`false`) — declare it explicitly if machine access is something your mod
can use.
