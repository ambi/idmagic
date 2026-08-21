---
name: spec-render
description: Compile TypeSpec and regenerate ignored OpenAPI and browsable specification HTML after specification or design changes.
---

# Regenerating derived specification artifacts

1. Run `mise run spec-render`.
2. Check for breakage against the release baseline with `mise run check-api-compat`.
3. Confirm that the OpenAPI carries per-context tags and that `spec/generated/docs/index.html` is
   produced.
4. `spec/generated/` is untracked. Do not commit it.
5. Update the baseline only during a release, with `mise run update-api-baseline`, as an explicit release
   step. Never update the baseline during an ordinary feature change to get around the compatibility
   check: skipping it lets a real regression ship, and running it without a release leaves the baseline
   describing something nobody received.
