# Environment Seed Profiles

Seeds are not executed by default during server startup. Plan and apply them explicitly using:
`just seed <environment> <profile> <mode> [manifest]`

Only `just dev` and `just dev-memory` explicitly apply the `development` profile in the same process for local convenience.

The desired state for a profile is located in `seed/manifests/*.yaml`. You can select a different root manifest via the fourth CLI argument or the `SEED_MANIFEST` environment variable. Manifests are strictly decoded and only local relative includes are allowed. Unknown keys, duplicate logical keys, circular references, paths outside the root, and YAML anchors/aliases/merges are rejected before any writes occur.

| Profile | Allowed environments | Contents |
| --- | --- | --- |
| `bootstrap` | development / test / staging / production | Minimal configuration for first-party clients. `SEED_FIRST_PARTY_REDIRECT_URIS` is required in production. |
| `development` | development / test / staging | Local demo user, group, protocol samples, and applications. |
| `test` | test | Deterministic fixtures identical to `development`. |
| `performance` | development / test / staging | Deterministic synthetic users. Typically up to 10,000 records; exceeding this requires the `--allow-large` flag. |

Review changes using `just seed development development dry_run`, then change the mode to `apply` to execute them. Seed output only includes logical keys and counts, omitting full passwords, client secrets, TOTP secrets, hashes, and PII.

Secret values must not be placed directly in YAML. Instead, they should be referenced using a `provider` (`env` or `file`), a `locator`, and a `version`. Staging and production environments only permit the `file` provider, and file locators are strictly limited to regular files under `SEED_SECRET_ROOT`. The `dry_run` mode also validates whether references can be resolved.

Because the old `SKIP_DEMO_SEED` flag has been removed, there is no need to configure startup settings to disable demo seeding. If a demo seed already exists in an environment, applying the same `development` profile will perform a semantic comparison, retaining manual changes as conflicts.

Performance profiles should not be included in normal verification flows. Use a small record count via `just seed development performance apply` for setup, and run `just seed-throughput development 10000 250` only during measurements. For counts exceeding 10,000, specify the `--allow-large` CLI flag; values over 100,000 are rejected.
