import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { TOOLS_DIR } from './workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

/**
 * 正本文書として不正な本文。H1 が 2 つある。名前が許可リストに載っていれば
 * `check-specifications.ts` が落とすので、この本文を持つファイルが通ったという
 * ことは、そのファイルが検証の対象にすら入っていないということである。
 */
const INVALID_BODY = '# One\n\n# Two\n'
const DEMO_SCENARIO = [
  '# Feature: Demo Scenarios',
  '',
  '## Rule: REQ-DEMO-001 Demo succeeds',
  '',
  '### Example: EX-DEMO-001-01 valid request',
  '',
  '- When the user submits a request',
  '- Then the request succeeds',
  '',
].join('\n')

/** 仮の作業ツリー。正本文書の集合が閉じているかどうかだけを見る最小の形。 */
async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-workspace-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'docs', 'contexts', 'demo'), { recursive: true })
  // Context を 1 つでも持つ作業ツリーは、索引表でその区分を宣言しなければならない。
  await writeFile(
    join(root, 'docs', 'README.md'),
    [
      '# Specification',
      '',
      '| Specification context | Subdomain | Go package | Responsibility |',
      '| --- | --- | --- | --- |',
      '| [Demo](contexts/demo/README.md) | Core | `demo` | Demo. |',
      '',
    ].join('\n'),
  )
  await writeFile(join(root, 'docs', 'contexts', 'demo', 'README.md'), '# Demo\n')
  await writeFile(
    join(root, 'docs', 'contexts', 'demo', 'scenarios.feature.md'),
    '# Feature: Demo Scenarios\n',
  )
  return root
}

/** `--documents` を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkDocuments(
  root: string,
  ...options: string[]
): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    [
      'bun',
      'run',
      resolve(TOOLS_DIR, 'workspace/src/check-workspace.ts'),
      '--documents',
      ...options,
    ],
    {
      cwd: TOOLS_DIR,
      env: { ...process.env, SPEC_WORKSPACE_ROOT: root },
      stdout: 'pipe',
      stderr: 'pipe',
    },
  )
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { code, output: `${stdout}${stderr}` }
}

/** `--work-items` を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkWorkItems(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    ['bun', 'run', resolve(TOOLS_DIR, 'workspace/src/check-workspace.ts'), '--work-items'],
    {
      cwd: TOOLS_DIR,
      env: { ...process.env, SPEC_WORKSPACE_ROOT: root },
      stdout: 'pipe',
      stderr: 'pipe',
    },
  )
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { code, output: `${stdout}${stderr}` }
}

describe('check-workspace --documents', () => {
  it('accepts a directory whose Markdown files are all canonical documents', async () => {
    const result = await checkDocuments(await workspace())
    expect(result.output).toContain('ok  3 canonical document(s)')
    expect(result.code).toBe(0)
  })

  /**
   * 成功した対象を 1 件ずつ並べるのは既定ではやらない。エージェントはこの出力を読んで
   * 文脈に載せるので、行数はそのまま所要時間と文脈の消費になる。全件は要求されたときだけ出す。
   */
  it('lists every passing document only when asked', async () => {
    const root = await workspace()
    expect((await checkDocuments(root)).output).not.toContain('scenarios.feature.md')
    expect((await checkDocuments(root, '--verbose')).output).toContain(
      'docs/contexts/demo/scenarios.feature.md',
    )
  })

  it('rejects a Markdown file the closed set does not name', async () => {
    const root = await workspace()
    await writeFile(join(root, 'docs', 'decision.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/decision.md')
  })

  it('names the canonical document a misspelled file was meant to be', async () => {
    const root = await workspace()
    await writeFile(join(root, 'docs', 'contexts', 'demo', 'scenario.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('scenarios.feature.md')
  })

  // 名前を全部打ち間違えた作業ツリー。集めた文書が 0 件になるという形で現れるので、
  // 検査を文書の件数で条件付けていると、この最悪の場合だけ黙って通る。
  it('rejects misspelled documents even when no canonical document is found at all', async () => {
    const root = await mkdtemp(join(tmpdir(), 'check-workspace-test-'))
    cleanup.push(root)
    await mkdir(join(root, 'docs'), { recursive: true })
    await mkdir(join(root, 'work-items'), { recursive: true })
    await writeFile(join(root, 'docs', 'readme.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('did you mean README.md')
  })

  it('leaves the freely named directories below the closed set alone', async () => {
    const root = await workspace()
    await mkdir(join(root, 'docs', 'runbooks'), { recursive: true })
    await mkdir(join(root, 'docs', 'development'), { recursive: true })
    await writeFile(join(root, 'docs', 'runbooks', 'anything.md'), INVALID_BODY)
    await writeFile(join(root, 'docs', 'development', 'release.md'), INVALID_BODY)

    expect((await checkDocuments(root)).code).toBe(0)
  })

  // 被覆のゲートは、負債台帳を持たない作業ツリーでこそ厳しい側へ倒れなければ
  // ならない。台帳が無いことを「負債を許さない」と読まずに素通りさせると、
  // 台帳を消すだけでゲートが外れる。
  it('rejects a scenario no test names when no debt list admits it', async () => {
    const root = await workspace()
    await writeFile(
      join(root, 'docs', 'contexts', 'demo', 'scenarios.feature.md'),
      [
        '# Feature: Demo Scenarios',
        '',
        '## Rule: REQ-DEMO-001 A valid request succeeds',
        '',
        '### Example: EX-DEMO-001-01 valid request',
        '',
        '- When the user submits a request',
        '- Then the request succeeds',
        '',
      ].join('\n'),
    )

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('EX-DEMO-001-01 is declared, but no test names it')
  })

  it('leaves a retired scenario out of the coverage gate', async () => {
    const root = await workspace()
    await writeFile(
      join(root, 'docs', 'contexts', 'demo', 'scenarios.feature.md'),
      [
        '# Feature: Demo Scenarios',
        '',
        '## Rule: REQ-DEMO-001 A valid request succeeds (superseded by REQ-DEMO-002)',
        'Replaced by the request-scoped route.',
        '',
        '## Rule: REQ-DEMO-002 A valid request succeeds',
        '',
        '### Example: EX-DEMO-002-01 valid request',
        '',
        '- When the user submits a request',
        '- Then the request succeeds',
        '',
      ].join('\n'),
    )
    await mkdir(join(root, 'backend'), { recursive: true })
    await writeFile(join(root, 'backend', 'demo_test.go'), 'package demo\n\n// EX-DEMO-002-01\n')

    expect((await checkDocuments(root)).code).toBe(0)
  })
})

describe('check-workspace --work-items', () => {
  it('rejects a completed item whose required release note is missing', async () => {
    const root = await workspace()
    await mkdir(join(root, 'work-items'), { recursive: true })
    await writeFile(
      join(root, 'work-items', 'wi-452-missing-release-note.md'),
      `---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-31
change_kind: tooling
evidence_policy: risk-based-v3
spec_impact: { kind: none, reason: "The fixture changes repository tooling only." }
documentation_impact:
  level: release_note
  reason: The declared release note must exist when the item is complete.
  references:
    - { kind: release_note, path: docs/releases/changes/wi-452-missing-release-note.md }
---

# Missing required release note

## Motivation

Exercise the documentation gate.

## Scope

- Work item validation

## Out of Scope

- Product behavior

## Verification

- mise run verify

## Risk Notes

The gate must reject an absent document.

## Completion

- **Completed At**: 2026-08-31
- **Summary**: The fixture declares a release note that is absent.
- **Acceptance RED Evidence**:
  - **Test**: check-workspace rejects a missing release note
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the incomplete fixture was accepted
  - **Detection Reason**: the CLI must resolve the planned path at completion
- **Unit RED Evidence**:
  - **Test**: verifyDocumentationImpact rejects a missing release note
  - **Requirement**: N/A: repository tooling has no normative product requirement
  - **Observed Failure**: the pure verifier was absent
  - **Detection Reason**: the verifier distinguishes a declared path from an existing document
- **Verification Results**:
  - mise run verify - passed
`,
    )

    const result = await checkWorkItems(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain(
      'documentation reference does not exist: docs/releases/changes/wi-452-missing-release-note.md',
    )
  })

  it('rejects an applicable in-progress item without a primary-use-case plan', async () => {
    const root = await workspace()
    await mkdir(join(root, 'work-items'), { recursive: true })
    await writeFile(join(root, 'docs', 'contexts', 'demo', 'scenarios.feature.md'), DEMO_SCENARIO)
    await writeFile(
      join(root, 'work-items', 'wi-439-missing-primary-use-case.md'),
      `---
status: in_progress
authors: [tn]
risk: medium
created_at: 2026-08-30
depends_on: []
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: The planned feature is noteworthy to release readers.
  references:
    - { kind: release_note, path: docs/releases/changes/wi-439-missing-primary-use-case.md }
initial_context:
  source: [docs/contexts/demo/scenarios.feature.md]
affected_spec:
  - { path: docs/contexts/demo/scenarios.feature.md, requirement: REQ-DEMO-001 }
---

# Feature without a primary use case

## Motivation

Exercise the evidence gate.

## Scope

- Demo feature

## Out of Scope

- Other features

## Verification

- mise run verify

## Risk Notes

The feature could remain disconnected.
`,
    )

    const result = await checkWorkItems(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('primary_use_cases')
  })

  it('accepts complete primary evidence only when both tests are reached by a required task', async () => {
    const root = await workspace()
    await mkdir(join(root, 'work-items'), { recursive: true })
    await mkdir(join(root, 'backend', 'demo'), { recursive: true })
    await writeFile(join(root, 'docs', 'contexts', 'demo', 'scenarios.feature.md'), DEMO_SCENARIO)
    await writeFile(
      join(root, 'mise.toml'),
      '[tasks.verify]\ndepends = ["test-go-race"]\n\n[tasks.test-go-race]\nrun = "go test -race ./..."\n',
    )
    await writeFile(
      join(root, 'backend', 'demo', 'rule_test.go'),
      'func TestDemoRule_REQ_DEMO_001(t *testing.T) { /* REQ-DEMO-001 */ }\n',
    )
    await writeFile(
      join(root, 'backend', 'demo', 'e2e_test.go'),
      'func TestE2E_Demo_REQ_DEMO_001(t *testing.T) { /* REQ-DEMO-001 */ }\n',
    )
    const workItem = (task: string): string => `---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-30
change_kind: feature
evidence_policy: risk-based-v3
affected_spec:
  - { path: docs/contexts/demo/scenarios.feature.md, requirement: REQ-DEMO-001 }
primary_use_cases:
  - id: demo-success
    requirement: REQ-DEMO-001
    observable_result: The caller observes the completed demo effect.
    unit_test: { path: backend/demo/rule_test.go, name: TestDemoRule_REQ_DEMO_001, task: ${task} }
    e2e_test: { path: backend/demo/e2e_test.go, name: TestE2E_Demo_REQ_DEMO_001, task: ${task} }
    unit_fault_model: The use case skips the effect.
    e2e_fault_model: The route is disconnected.
---

# Feature with primary evidence

## Motivation

Exercise completed evidence.

## Scope

- Demo feature

## Out of Scope

- Other features

## Verification

- mise run verify

## Risk Notes

The feature could remain disconnected.

## Completion

- **Completed At**: 2026-08-30
- **Summary**: The demo route now produces its final effect.
- **Primary Use Case Evidence**:
  - id: demo-success
    unit_red: the unit test observed no effect
    e2e_red: the E2E test observed no final result
    unit_fault_injection: removing the effect made the unit test fail
    e2e_fault_injection: disconnecting the route made the E2E test fail
- **Verification Results**:
  - mise run verify - passed
`
    const path = join(root, 'work-items', 'wi-445-complete-primary-use-case.md')
    await writeFile(path, workItem('test-go-race'))
    expect((await checkWorkItems(root)).code).toBe(0)

    await writeFile(path, workItem('test-go'))
    const unreachable = await checkWorkItems(root)
    expect(unreachable.code).not.toBe(0)
    expect(unreachable.output).toContain('task is not required by verify or CI: test-go')
  })
})
