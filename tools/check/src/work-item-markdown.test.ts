import { afterEach, describe, expect, it } from 'bun:test'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const temporaryDirectories: string[] = []

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })))
})

describe('work-item Markdown completion evidence', () => {
  it('parses structured RED and stronger completion evidence', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'idmagic-work-item-'))
    temporaryDirectories.push(directory)
    const path = join(directory, 'wi-410-parser-evidence.md')
    await writeFile(
      path,
      `---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
evidence_policy: risk-based-v1
approval:
  by: tn
  at: 2026-08-23
  scope: Parse completion evidence.
  baseline: 3cb041f1d61007a3213ead7c1bba989d1d19824a
---

# Parse completion evidence

## Motivation

Keep evidence machine-readable.

## Scope

- Markdown parser

## Out of Scope

- Product behavior

## Verification

- mise run test-tools

## Risk Notes

The parser must reject incomplete evidence.

## Completion

- **Completed At**: 2026-08-23
- **Summary**: Parsed the evidence.
- **RED Evidence**:
  - **Test**: parser rejects missing evidence
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete record failed validation
  - **Detection Reason**: each nested field maps to a required schema property
- **Post-Approval Changes**: none
- **Independent Verification**: reviewed by another agent
- **Change-Resistance Results**: removing one field fails validation
- **Verification Results**:
  - mise run test-tools - passed
`,
    )

    const child = Bun.spawn(
      [process.execPath, 'run', resolve(import.meta.dir, 'main.ts'), '--schema=work-item', path],
      { stderr: 'pipe', stdout: 'pipe' },
    )
    const [exitCode, stdout, stderr] = await Promise.all([
      child.exited,
      new Response(child.stdout).text(),
      new Response(child.stderr).text(),
    ])

    expect(exitCode, stderr).toBe(0)
    expect(stdout).toContain('ok')
  })
})
