---
name: render-docs
description: Compile TypeSpec and regenerate the ignored OpenAPI and the browsable specification site after specification or design changes; regenerate the tracked route metadata when TypeSpec operations change.
---

# Regenerating derived specification artifacts

1. Run `mise run render-docs`. It compiles TypeSpec first, so the site never renders a stale OpenAPI.
2. When the change adds, removes, or alters a TypeSpec operation, also run `mise run generate-contract`.
   It rewrites the tracked `backend/shared/spec/operations_gen.go`, which `check-generated-contract`
   compares against TypeSpec; commit that file with the change.
3. `mise run check-published-api-compat` で公開状態に応じた互換性を検査する。
   初回公開前の差分を任意に調べる場合は `mise run check-api-compat` を使う。
   公開状態と初回公開の切り替えは `docs/development/release.md` が定める。
4. Confirm that the OpenAPI carries per-module tags and that `site/index.html` is
   produced.
5. `spec/generated/` and `site/` are untracked. Do not commit them.
6. Update the baseline only during a release, with `mise run update-api-baseline`, as an explicit release
   step. Never update the baseline during an ordinary feature change to get around the compatibility
   check: skipping it lets a real regression ship, and running it without a release leaves the baseline
   describing something nobody received.
