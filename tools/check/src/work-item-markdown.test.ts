import { afterEach, describe, expect, it } from 'bun:test'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { parseFrontmatterAndMarkdown } from './main.ts'

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

  it('parses separate Acceptance RED and Unit RED evidence', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'idmagic-work-item-'))
    temporaryDirectories.push(directory)
    const path = join(directory, 'wi-412-parser-evidence.md')
    await writeFile(
      path,
      `---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
evidence_policy: risk-based-v2
---

# Parse separate RED evidence

## Motivation

Keep both evidence boundaries machine-readable.

## Scope

- Markdown parser

## Out of Scope

- Product behavior

## Verification

- mise run test-tools

## Risk Notes

The parser must reject either missing boundary.

## Completion

- **Completed At**: 2026-08-23
- **Summary**: Parsed both evidence boundaries.
- **Acceptance RED Evidence**:
  - **Test**: parser rejects missing acceptance evidence
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete record failed validation
  - **Detection Reason**: the acceptance field is independently required
- **Unit RED Evidence**:
  - **Test**: parser rejects missing unit evidence
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete record failed validation
  - **Detection Reason**: the unit field is independently required
- **Independent Verification**: reviewed by another agent
- **Change-Resistance Results**: removing either field fails validation
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

  it('parses primary-use-case completion evidence as structured YAML', () => {
    const source = `---
status: completed
---

# Primary use-case evidence

## Completion

- **Completed At**: 2026-08-30
- **Summary**: Recorded primary use-case evidence.
- **Primary Use Case Evidence**:
  - id: configured-delivery
    unit_red: the use-case test observed no outgoing effect
    e2e_red: the external entry produced no delivery
    unit_fault_injection: removing the branch made the unit test fail
    e2e_fault_injection: disconnecting the adapter made the E2E test fail
`

    expect(parseFrontmatterAndMarkdown('wi-445-demo.md', source).completion).toMatchObject({
      primary_use_case_evidence: [
        {
          id: 'configured-delivery',
          unit_red: 'the use-case test observed no outgoing effect',
          e2e_red: 'the external entry produced no delivery',
          unit_fault_injection: 'removing the branch made the unit test fail',
          e2e_fault_injection: 'disconnecting the adapter made the E2E test fail',
        },
      ],
    })
  })
})
