import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { TOOLS_DIR } from '../../workspace/src/workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

/**
 * 一次情報文書として不正な本文。H1 が 2 つある。名前が許可リストに載っていれば
 * 文書検査が落とすので、この本文を持つファイルが通ったという
 * ことは、そのファイルが検証の対象にすら入っていないということである。
 */
const INVALID_BODY = '# One\n\n# Two\n'
/** 機能スライス `run` の機能仕様。work item が参照する要件を一つ宣言する。 */
const DEMO_SPECIFICATION = '# 実行\n\n#### REQ-DEMO-001 Demo succeeds\n\n- 要求を受け付ける。\n'

/** 正常な要求が成功するという要件と、その例。どのテストも例を名指さない。 */
const VALID_REQUEST_SPECIFICATION =
  '# 実行\n\n#### REQ-DEMO-001 A valid request succeeds\n\n- 正常な要求は成功する。\n'
const VALID_REQUEST_EXAMPLES = [
  '# Feature: 実行の例',
  '',
  '## Rule: REQ-DEMO-001 A valid request succeeds',
  '',
  '### Example: EX-DEMO-001-01 valid request',
  '',
  '- When the user submits a request',
  '- Then the request succeeds',
  '',
].join('\n')

/** 設計の入口。どの話題にも固有の設計がないことを、理由とともに書く。 */
const DESIGN_INDEX = [
  '# Demo の設計',
  '',
  '| 話題 | 記述した場所 |',
  '| --- | --- |',
  ...[
    'アーキテクチャ',
    '設計判断',
    'アプリケーション',
    'データ',
    'セキュリティ',
    '信頼性',
    '性能',
    'オブザーバビリティ',
    '検証',
    'インフラストラクチャ',
    'リスク',
  ].map((topic) => `| ${topic} | 該当なし：Demo は小さく、固有の設計がない |`),
  '',
].join('\n')

/**
 * 機能スライスに、要件を宣言する機能仕様と例の付録を置く。`node` はモジュールからの相対パスで、
 * 機能群の下の機能スライスも指せる。
 */
async function writeFeature(
  root: string,
  node: string,
  specification: string,
  examples: string,
): Promise<void> {
  const directory = join(root, 'docs', 'domain', 'demo', ...node.split('/'))
  await mkdir(directory, { recursive: true })
  await writeFile(join(directory, 'README.md'), specification)
  await writeFile(join(directory, 'acceptance.feature.md'), examples)
}

/** 仮の作業ツリー。一次情報文書の集合が閉じているかどうかだけを見る最小の形。 */
async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-workspace-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'docs', 'domain', 'demo', 'design'), { recursive: true })
  await mkdir(join(root, 'docs', 'design', 'architecture'), { recursive: true })
  await writeFile(join(root, 'docs', 'README.md'), '# Specification\n')
  await writeFile(
    join(root, 'docs', 'design', 'architecture', 'logical.md'),
    [
      '# 論理アーキテクチャ',
      '',
      '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |',
      '| --- | --- | --- | --- | --- |',
      '| [Demo](../../domain/demo/README.md) | `legacy` | `domain` または `ports` の区画 | `demo` | Demo. |',
      '',
    ].join('\n'),
  )
  await writeFile(join(root, 'docs', 'domain', 'demo', 'README.md'), '# Demo\n')
  await writeFile(join(root, 'docs', 'domain', 'demo', 'design', 'README.md'), DESIGN_INDEX)
  return root
}

/** 文書検査を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkDocuments(
  root: string,
  ...options: string[]
): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    ['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), 'documents', ...options],
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

/** 文書配置図と定義済み文書の整合検査を仮の作業ツリーに対して起動する。 */
async function checkDocumentLayout(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    ['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), 'document-layout'],
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

/** 用語検査を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkTerminology(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), 'terminology'], {
    cwd: TOOLS_DIR,
    env: { ...process.env, SPEC_WORKSPACE_ROOT: root },
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { code, output: `${stdout}${stderr}` }
}

/** work item 参照の検査を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkWorkItemReferences(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    ['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), 'work-item-references'],
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

/** work item 検査を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkWorkItems(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), 'work-items'], {
    cwd: TOOLS_DIR,
    env: { ...process.env, SPEC_WORKSPACE_ROOT: root },
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { code, output: `${stdout}${stderr}` }
}

describe('文書検査', () => {
  it('accepts a directory whose Markdown files are all canonical documents', async () => {
    const result = await checkDocuments(await workspace())
    expect(result.output).toContain('ok  4 canonical document(s)')
    expect(result.code).toBe(0)
  })

  /**
   * 成功した対象を 1 件ずつ並べるのは既定ではやらない。エージェントはこの出力を読んで
   * 文脈に載せるので、行数はそのまま所要時間と文脈の消費になる。全件は要求されたときだけ出す。
   */
  it('lists every passing document only when asked', async () => {
    const root = await workspace()
    expect((await checkDocuments(root)).output).not.toContain('docs/domain/demo/design/README.md')
    const verbose = await checkDocuments(root, '--verbose')
    expect(verbose.output).toContain('docs/domain/demo/design/README.md')
    expect(verbose.code).toBe(0)
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
    await writeFile(join(root, 'docs', 'domain', 'demo', 'glosary.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('did you mean glossary.md?')
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
    await writeFeature(root, 'run', VALID_REQUEST_SPECIFICATION, VALID_REQUEST_EXAMPLES)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('EX-DEMO-001-01 is declared, but no test names it')
  })

  // 機能群の下の機能スライスの文書が一次情報として読まれなければ、そこへ置いた要件は宣言ごと
  // 検査から消える。被覆の拒否が出ることが、読まれていることの観測になる。
  it('reads the examples of a feature slice inside a group as normative', async () => {
    const root = await workspace()
    await mkdir(join(root, 'docs', 'domain', 'demo', 'people'), { recursive: true })
    await writeFile(join(root, 'docs', 'domain', 'demo', 'people', 'README.md'), '# People\n')
    await writeFeature(
      root,
      'people/user',
      '# User\n\n#### REQ-DEMO-002 A user is created on request\n\n- 要求でユーザーを作る。\n',
      [
        '# Feature: User',
        '',
        '## Rule: REQ-DEMO-002 A user is created on request',
        '',
        '### Example: EX-DEMO-002-01 valid request',
        '',
        '- When the user submits a request',
        '- Then the user exists',
        '',
      ].join('\n'),
    )

    const result = await checkDocuments(root)
    expect(result.output).not.toContain('not a canonical specification document')
    expect(result.output).not.toContain('docs/domain/demo/people/user/ is not listed')
    expect(result.output).toContain('EX-DEMO-002-01 is declared, but no test names it')
  })

  // 標準の側の台帳は wi-495 が空にして消した。消えたのは一覧だけでなく免除の
  // 仕組みそのものなので、同じ名前のファイルを置き直しても行は通らない。これが
  // 成り立たなければ、台帳の削除は誰でも元に戻せる。
  it('admits no standards row through a recreated debt ledger', async () => {
    const root = await workspace()
    await writeFile(
      join(root, 'docs', 'domain', 'demo', 'standards.md'),
      [
        '# Demo の採用規範',
        '',
        '## Demo Protocol',
        '',
        'RFC DEMO — https://example.com/rfc-demo',
        '',
        '| ID | Adoption | Strength | Statement |',
        '|---|---|---|---|',
        '| RFC-DEMO-001 | required | MUST | デモは要求を受け付ける。 |',
        '',
      ].join('\n'),
    )
    await mkdir(join(root, 'tools', 'check'), { recursive: true })
    await writeFile(
      join(root, 'tools', 'check', 'standards-coverage-debt.json'),
      JSON.stringify({ untested: [{ id: 'RFC-DEMO-001', reason: 'ここへ書けば通ると思った' }] }),
    )

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('RFC-DEMO-001 is declared, but no test names it')
    // 逃げ道を案内しない。存在しない手順へ読み手を送らないための表明でもある。
    expect(result.output).not.toContain('standards-coverage-debt.json')
  })

  // 具体例の台帳も wi-496 が空にして消し、免除の仕組みはリポジトリから無くなった。
  // 同じ名前のファイルを置き直して通るなら、台帳の削除は 1 ファイルで元に戻せる。
  it('admits no example through a recreated debt ledger', async () => {
    const root = await workspace()
    await writeFeature(root, 'run', VALID_REQUEST_SPECIFICATION, VALID_REQUEST_EXAMPLES)
    await mkdir(join(root, 'tools', 'check'), { recursive: true })
    await writeFile(
      join(root, 'tools', 'check', 'example-coverage-debt.json'),
      JSON.stringify({ untested: [{ id: 'EX-DEMO-001-01', reason: 'ここへ書けば通ると思った' }] }),
    )

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('EX-DEMO-001-01 is declared, but no test names it')
    expect(result.output).not.toContain('example-coverage-debt.json')
  })

  it('leaves a retired scenario out of the coverage gate', async () => {
    const root = await workspace()
    await writeFeature(
      root,
      'run',
      [
        '# 実行',
        '',
        '#### REQ-DEMO-001 A valid request succeeds (superseded by REQ-DEMO-002)',
        '',
        '- 要求ごとの経路へ置き換えた。',
        '',
        '#### REQ-DEMO-002 A valid request succeeds',
        '',
        '- 正常な要求は成功する。',
        '',
      ].join('\n'),
      VALID_REQUEST_EXAMPLES.replaceAll('DEMO-001', 'DEMO-002'),
    )
    await mkdir(join(root, 'backend'), { recursive: true })
    await writeFile(
      join(root, 'backend', 'demo_test.go'),
      'package demo\n\n//spec:covers EX-DEMO-002-01: 正常な要求が通ることを固定する。\n',
    )

    expect((await checkDocuments(root)).code).toBe(0)
  })
})

describe('文書配置図の整合検査', () => {
  it('登録した文書配置検査が配置図の欠落を拒否する', async () => {
    const root = await mkdtemp(join(tmpdir(), 'check-document-layout-test-'))
    cleanup.push(root)
    await mkdir(join(root, 'docs/formats'), { recursive: true })
    await writeFile(
      join(root, 'docs/formats/specification-format.md'),
      '# 仕様フォーマット\n\n## 1. 配置\n\n```text\ndocs/\n  README.md\n```\n',
    )

    const result = await checkDocumentLayout(root)

    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/domain/<context>/standards.md')
  })
})

describe('work item 検査', () => {
  it('rejects the same record in pending and done', async () => {
    const root = await workspace()
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    const record = `---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-11
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "The fixture changes repository tooling only." }
---

# Duplicate record fixture

## Motivation

Exercise work-item collection integrity.

## Scope

- Work-item validation

## Out of Scope

- Product behavior

## Verification

- mise run test-tools

## Risk Notes

The fixture must remain otherwise valid.
`
    const name = 'wi-531-duplicate-record.md'
    await writeFile(join(root, 'work-items', name), record)
    await writeFile(join(root, 'work-items', 'done', name), record)

    const result = await checkWorkItems(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain(`duplicate work item '${name.replace(/\.md$/, '')}'`)
  })

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
    await mkdir(join(root, 'docs', 'domain', 'demo', 'run'), { recursive: true })
    await writeFile(join(root, 'docs', 'domain', 'demo', 'run', 'README.md'), DEMO_SPECIFICATION)
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
  source: [docs/domain/demo/run/README.md]
affected_spec:
  - { path: docs/domain/demo/run/README.md, requirement: REQ-DEMO-001 }
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
    await mkdir(join(root, 'docs', 'domain', 'demo', 'run'), { recursive: true })
    await writeFile(join(root, 'docs', 'domain', 'demo', 'run', 'README.md'), DEMO_SPECIFICATION)
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
  - { path: docs/domain/demo/run/README.md, requirement: REQ-DEMO-001 }
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

describe('用語検査', () => {
  it('採らない表記を含む設計文書を、行と採用語つきで拒否する', async () => {
    const root = await workspace()
    await writeFile(
      join(root, 'docs', 'design', 'architecture', 'deployment.md'),
      '# 概要\n\n配備の不変条件。\n',
    )

    const result = await checkTerminology(root)

    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/design/architecture/deployment.md:3:1')
    expect(result.output).toContain('デプロイ')
  })

  it('docs/formats 配下の文書も対象にする', async () => {
    const root = await workspace()
    await mkdir(join(root, 'docs/formats'), { recursive: true })
    await writeFile(
      join(root, 'docs/formats/documentation-guide.md'),
      '# 文書体系\n\n観測可能性の設計。\n',
    )

    const result = await checkTerminology(root)

    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/formats/documentation-guide.md:3:1')
    expect(result.output).toContain('オブザーバビリティ')
  })

  it('root 直下のエージェント指示も対象にする', async () => {
    const root = await workspace()
    await writeFile(join(root, 'AGENTS.md'), '# エージェント指示\n\n観測可能性の設計。\n')

    const result = await checkTerminology(root)

    expect(result.code).not.toBe(0)
    expect(result.output).toContain('AGENTS.md:3:1')
  })

  it('採用語だけの作業ツリーを通す', async () => {
    const result = await checkTerminology(await workspace())

    expect(result.code).toBe(0)
    expect(result.output).toContain('ok  terminology')
  })
})

describe('現在状態の文書からの work item 参照の検査', () => {
  it('設計文書の work item 参照を、位置つきで拒否する', async () => {
    const root = await workspace()
    await writeFile(
      join(root, 'docs', 'design', 'architecture', 'deployment.md'),
      '# 概要\n\n試験は [wi-165](../../work-items/wi-165-ha.md) が扱う。\n',
    )

    const result = await checkWorkItemReferences(root)

    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/design/architecture/deployment.md:3:6')
    expect(result.output).toContain('wi-165')
  })

  it('work item を参照しない作業ツリーを通す', async () => {
    const result = await checkWorkItemReferences(await workspace())

    expect(result.code).toBe(0)
    expect(result.output).toContain('ok  work-item-references')
  })
})

/** 規則の書式と仕様の木の検査を仮の作業ツリーに対して起動する。 */
async function checkSpecificationRules(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    ['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), 'specification-rules'],
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

describe('規則の書式と仕様の木の検査', () => {
  /** 機能スライス `demo/task` の、コードのディレクトリと仕様のディレクトリを持つ作業ツリー。 */
  async function featureWorkspace(guarantee: string, parent: string): Promise<string> {
    const root = await workspace()
    await mkdir(join(root, 'backend', 'demo', 'task', 'domain'), { recursive: true })
    await writeFile(
      join(root, 'backend', 'demo', 'task', 'domain', 'task.go'),
      'package domain\n\ntype Task struct{}\n\nfunc (t Task) Open() bool { return true }\n',
    )
    await mkdir(join(root, 'docs', 'design', 'application'), { recursive: true })
    await writeFile(
      join(root, 'docs', 'design', 'application', 'api-guidelines.md'),
      '# API ガイドライン\n\n## ページサイズ\n\n既定は 50 件とする。\n',
    )
    await mkdir(join(root, 'docs', 'domain', 'demo', 'task'), { recursive: true })
    await writeFile(
      join(root, 'docs', 'domain', 'demo', 'task', 'README.md'),
      [
        '# タスク',
        '',
        '## 操作',
        '',
        '### タスクの一覧',
        '',
        '#### REQ-DEMO-002 開いたタスクだけを一覧する',
        '',
        '- 既定値は 10 件とする。',
        `- **上位の要件**：${parent}`,
        `- **担保手段**：\`${guarantee}\``,
        '',
      ].join('\n'),
    )
    return root
  }
  const parentLink = '[ページサイズ](../../../design/application/api-guidelines.md#ページサイズ)'

  it('accepts a feature slice whose guarantee and parent requirement both resolve', async () => {
    const result = await checkSpecificationRules(await featureWorkspace('Task.Open', parentLink))
    expect(result.output).toContain('ok  specification rules')
    expect(result.code).toBe(0)
  })

  it('rejects a guarantee the code does not declare', async () => {
    const result = await checkSpecificationRules(await featureWorkspace('Task.Close', parentLink))
    expect(result.code).not.toBe(0)
    expect(result.output).toContain(
      'REQ-DEMO-002 guarantee `Task.Close` is not declared in backend/',
    )
  })

  it('rejects a parent requirement whose heading does not exist', async () => {
    const result = await checkSpecificationRules(
      await featureWorkspace(
        'Task.Open',
        '[ページサイズ](../../../design/application/api-guidelines.md#上限)',
      ),
    )
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-002 parent requirement does not resolve')
  })

  it('rejects a code slice with no feature slice specification', async () => {
    const root = await featureWorkspace('Task.Open', parentLink)
    await mkdir(join(root, 'backend', 'demo', 'note', 'usecases'), { recursive: true })
    await writeFile(
      join(root, 'backend', 'demo', 'note', 'usecases', 'note.go'),
      'package usecases\n',
    )
    const result = await checkSpecificationRules(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain(
      'backend/demo/note has no feature slice specification under docs/domain/demo/',
    )
  })
})
