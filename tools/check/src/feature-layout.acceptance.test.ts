/**
 * 機能仕様と内部設計を軸にした形式のモジュールを、検査が受け入れ、崩れを拒否することを確かめる。
 *
 * モジュールの印は `docs/modules/<module>/README.md` であり、この作業ツリーはそれを持つモジュール `demo` だけで組む。
 */
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { TOOLS_DIR } from '../../workspace/src/workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

const SPECIFICATION = [
  '# タスク',
  '',
  '## 概要',
  '',
  'タスクを扱う。',
  '',
  '## 状態遷移',
  '',
  '### TaskLifecycle',
  '',
  '| State | Kind | Meaning |',
  '|---|---|---|',
  '| open | initial | 開いている |',
  '| closed | terminal | 閉じた |',
  '',
  '| From | Event | Guard | To | Effects |',
  '|---|---|---|---|---|',
  '| open | TaskClosed | — | closed |  |',
  '',
  '| State | 閉じる |',
  '|---|---|',
  '| open | → closed |',
  '| closed | 何もしない |',
  '',
  '## 操作',
  '',
  '### タスクの一覧',
  '',
  '#### REQ-DEMO-002 開いたタスクだけを一覧する',
  '',
  '- 既定値は 10 件とする。',
  '- **判断**：閉じたタスクは利用者の作業対象ではない。',
  '- **担保手段**：`Task.Open`',
  '',
].join('\n')

const EXAMPLES = [
  '# Feature: タスクの例',
  '',
  '## Rule: REQ-DEMO-002 開いたタスクだけを一覧する',
  '',
  '### Example: EX-DEMO-002-01 開いたタスク',
  '',
  '- When 一覧を要求する',
  '- Then 開いたタスクを返す',
  '',
].join('\n')

const DECISIONS = [
  '# Demo の重要な設計判断',
  '',
  '## タスクを物理削除しない',
  '',
  '### 背景',
  '',
  '監査記録がタスクを参照する。',
  '',
  '### 決定',
  '',
  '閉じた状態として残す。',
  '',
  '### 検討した代替案',
  '',
  '| 案 | 利点 | 欠点 | 採らない理由 |',
  '| --- | --- | --- | --- |',
  '| 物理削除 | 領域を使わない | 参照が壊れる | 監査記録を壊す |',
  '',
  '### 結果と再検討の条件',
  '',
  '監査記録がタスクを参照しなくなったら再検討する。',
  '',
  '### 関連する要件',
  '',
  'REQ-DEMO-002',
  '',
].join('\n')

/** モジュールの直下の、設計領域以外の文書の索引。 */
const MODULE_DOCUMENT_TABLE = [
  '| 文書 | 内容 |',
  '| --- | --- |',
  '| [用語集](glossary.md) | 語義 |',
  '| [設計判断](decisions.md) | 判断 |',
  '',
].join('\n')

/**
 * モジュールの README の最後の節「文書」に、最小の設計領域の索引とモジュールの文書だけを書く。
 * 共通設計の九行は再掲しない。
 */
function areaIndex(title: string, data: string, rows: string[] = []): string {
  return [
    `# ${title}`,
    '',
    '## 文書',
    '',
    '| 設計領域 | 文書 | 内容 |',
    '| --- | --- | --- |',
    `| データ設計 | ${data} | 集約 |`,
    ...rows,
    '',
    MODULE_DOCUMENT_TABLE,
  ].join('\n')
}

/** モジュールを一つだけ持ち、どの検査にも通る作業ツリー。 */
async function featureLayoutWorkspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-feature-layout-test-'))
  cleanup.push(root)
  const files: Record<string, string> = {
    'docs/README.md': '# Specification\n',
    'docs/design/architecture/logical.md': [
      '# 論理アーキテクチャ',
      '',
      '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |',
      '| --- | --- | --- | --- | --- |',
      '| [Demo](../../modules/demo/README.md) | `internal` | なし | `demo` | Demo. |',
      '',
    ].join('\n'),
    'docs/modules/demo/glossary.md': '# Demo の用語集\n',
    'docs/modules/demo/README.md': areaIndex('Demo', '[Demo のデータ](data.md#構成)'),
    'docs/modules/demo/data.md': '# Demo のデータ\n\n## 構成\nタスクを扱う。\n',
    'docs/modules/demo/decisions.md': DECISIONS,
    'docs/modules/demo/features/task.md': `${SPECIFICATION}\n## 実現方式\n\n### アーキテクチャ\n\nタスクを保存する。\n`,

    'docs/modules/demo/features/task.feature.md': EXAMPLES,
    'backend/demo/task/domain/task.go':
      'package domain\n\ntype Task struct{}\n\nfunc (t Task) Open() bool { return true }\n',
    'backend/demo/task/domain/task_test.go': [
      'package domain',
      '',
      '//spec:covers EX-DEMO-002-01: 開いたタスクを一覧に含める',
      'func TestTaskOpen(t *testing.T) {}',
      '',
    ].join('\n'),
  }
  for (const [path, content] of Object.entries(files)) {
    await mkdir(dirname(join(root, path)), { recursive: true })
    await writeFile(join(root, path), content)
  }
  return root
}

async function write(root: string, path: string, content: string): Promise<void> {
  await mkdir(dirname(join(root, path)), { recursive: true })
  await writeFile(join(root, path), content)
}

async function runCheck(root: string, name: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(['bun', 'run', resolve(TOOLS_DIR, 'check/src/runner.ts'), name], {
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

describe('機能仕様と内部設計の形式の文書検査', () => {
  it('accepts a context with feature specifications, an examples appendix, and design', async () => {
    const result = await runCheck(await featureLayoutWorkspace(), 'documents')
    expect(result.output).toContain('1 rule(s), 1 example(s)')
    expect(result.code).toBe(0)
  })

  it('rejects the per-kind files of the legacy layout for a feature', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/features/task.decisions.md', '# 判断\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/modules/demo/features/task.decisions.md')
  })

  it('accepts a requirement that has no example in the appendix', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      `${SPECIFICATION}\n#### REQ-DEMO-003 閉じたタスクを隠す\n\n- 閉じたタスクは返さない。\n`,
    )
    const result = await runCheck(root, 'documents')
    expect(result.output).not.toContain('REQ-DEMO-003')
    expect(result.code).toBe(0)
  })

  it('rejects an appendix rule the specification does not declare', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.feature.md',
      EXAMPLES.replaceAll('REQ-DEMO-002', 'REQ-DEMO-009').replaceAll('EX-DEMO-002', 'EX-DEMO-009'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-009 is not declared by the feature specification')
  })

  it('rejects an appendix rule whose title differs from the declaration', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.feature.md',
      EXAMPLES.replace('開いたタスクだけを一覧する', 'タスクを一覧する'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-002 title differs from the feature specification')
  })

  it('rejects a rule declared outside a feature specification', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/data.md',
      '# Demo のアーキテクチャ\n\n## 構成\n\n#### REQ-DEMO-004 作業を数える\n\n- 作業を数える。\n',
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain(
      'REQ-DEMO-004 must be declared in scenarios.feature.md or in a feature specification',
    )
  })

  it('rejects a decision that leaves out a part of the decision record', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/decisions.md',
      DECISIONS.replace('### 検討した代替案', '### 比較'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('タスクを物理削除しない must have the section 検討した代替案')
  })

  it('accepts the quality requirements allocated to a context', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/quality.md', '# Demo の品質要件\n')
    await write(
      root,
      'docs/modules/demo/README.md',
      `${areaIndex('Demo', '[構成](data.md#構成)')}| [品質要件](quality.md) | 割り当て |\n`,
    )
    const result = await runCheck(root, 'documents')
    expect(result.output).not.toContain('docs/modules/demo/quality.md')
    expect(result.code).toBe(0)
  })

  it('共通設計だけのモジュールには領域の列挙を要求しない', async () => {
    const root = await featureLayoutWorkspace()
    await rm(join(root, 'docs/modules/demo/data.md'))
    await write(
      root,
      'docs/modules/demo/README.md',
      `# Demo\n\n## 文書\n\n${MODULE_DOCUMENT_TABLE}`,
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).toBe(0)
  })

  it('既存の設計領域の文書を索引から消すと共通設計への継承として隠せない', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/README.md', '# Demo\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('must index its own データ設計')
  })

  /** 機能の仕組みは機能仕様の「実現方式」の節に書き、別のファイルへ分けない。 */
  it('rejects a separate design document for a feature', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.design.md',
      '# タスクの設計\n\n## 信頼性\n\n削除が途中で失敗したら、削除を再実行する。\n',
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/modules/demo/features/task.design.md')
  })

  it('rejects the realization section placed before the operations', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      SPECIFICATION.replace('## 操作', '## 実現方式\n\nタスクを保存する。\n\n## 操作'),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('section 操作 must come before 実現方式')
  })

  it('未知の領域を拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/README.md',
      areaIndex('Demo', '[構成](data.md#構成)').replace('データ設計 |', '運用 |'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('design area index names an unknown area 運用')
  })

  it('リンクでない文書と、文書のない行の理由の欠落を拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/README.md',
      areaIndex('Demo', 'データの文書', ['| 性能設計 | — | |']),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('must link to a local section')
    expect(result.output).toContain('must state a reason')
  })

  it('文書のない行の内容にリンクを置くことを拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/README.md',
      areaIndex('Demo', '[構成](data.md#構成)', ['| 性能設計 | — | [保存](data.md#構成)に従う |']),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('must not link from its content')
  })

  it('委譲先の存在しない節を文書検査で拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/README.md', areaIndex('Demo', '[構成](data.md#ない節)'))
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('ない節')
  })

  it('委譲先の存在しないファイルを文書検査で拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/README.md', areaIndex('Demo', '[構成](missing.md#構成)'))
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('relative link target does not exist')
  })

  it('宣言済みの品質要件から担当への参照がない場合を拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/requirements/quality.md',
      '# 品質要件\n\n| ID | 母集団 | 目標 | 指標 |\n| --- | --- | --- | --- |\n| SLO-DEMO-LATENCY | タスク | 目標 | 指標 |\n',
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('quality allocation is missing SLO-DEMO-LATENCY')
  })

  it('品質要件から担当機能、設計、検証へ到達でき、存在しない担当の節を拒否する', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/requirements/quality.md',
      '# 品質要件\n\n## 目標\n\n| ID | 母集団 | 目標 | 指標 |\n| --- | --- | --- | --- |\n| SLO-DEMO-LATENCY | タスク | 目標 | 指標 |\n',
    )
    const path = 'docs/design/architecture/logical.md'
    const source =
      '# 論理アーキテクチャ\n\n| 品質要件 | 担当機能 | 設計 | 検証 |\n| --- | --- | --- | --- |\n| [SLO-DEMO-LATENCY](../../requirements/quality.md#目標) | [担当](../../modules/demo/features/task.md#操作) | [設計](../../modules/demo/data.md#構成) | [検証](../../modules/demo/features/task.md#アーキテクチャ) |\n'
    await write(root, path, source)
    expect((await runCheck(root, 'documents')).code).toBe(0)
    await write(root, path, source.replace('task.md#操作', 'task.md#ない操作'))
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('ない操作')
  })

  it('applies the design area index to the system design', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/design/README.md', '# 全体設計文書\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/design/README.md')
    expect(result.output).toContain('must have a design area index')
  })

  it('requires every module to have README.md', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/other/glossary.md', '# Other の用語集\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('other has no README.md')
  })

  /** 設計領域の文書はモジュールの直下に置き、`design/` の段を置かない。 */
  it('rejects a design directory below a module', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/design/README.md', '# Demo の設計\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/modules/demo/design/README.md')
  })

  it('rejects a design area index outside the last section 文書', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/README.md',
      `${areaIndex('Demo', '[構成](data.md#構成)')}\n## 機能\n\nタスクを扱う。\n`,
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('design area index must be in the last section 文書')
  })

  it('rejects the per-kind files of the former layout directly under a context', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/modules/demo/scenarios.feature.md', '# Feature: Demo\n')
    await write(root, 'docs/modules/demo/states.md', '# Demo の状態\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/modules/demo/scenarios.feature.md')
    expect(result.output).toContain('docs/modules/demo/states.md')
  })
})

describe('機能仕様の規則の書式', () => {
  it('accepts a rule with a guarantee and a decision note', async () => {
    const result = await runCheck(await featureLayoutWorkspace(), 'specification-rules')
    expect(result.output).toContain('ok  specification rules')
    expect(result.code).toBe(0)
  })

  it('accepts a requirement without a guarantee', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      SPECIFICATION.replace('- **担保手段**：`Task.Open`\n', ''),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.output).toContain('ok  specification rules')
    expect(result.code).toBe(0)
  })

  /** 共通シナリオはモジュールの段の外にあるが、機能仕様と同じ規則の書式で検査する。 */
  it('checks the rule fields of the shared scenarios beside the modules', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/scenarios.feature.md',
      [
        '# Feature: Cross-Module Scenarios',
        '',
        '## Rule: REQ-PLATFORM-001 共通の保証',
        '',
        '- **担保手段**：`Missing.Guard`',
        '',
      ].join('\n'),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-PLATFORM-001 guarantee `Missing.Guard` is not declared')
  })

  it('rejects the legacy reason field', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      SPECIFICATION.replace('**判断**', '**理由**'),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-002 uses 理由; write the reason as 判断')
  })

  it('rejects the retired parent field name', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      SPECIFICATION.replace(
        '- **担保手段**：`Task.Open`\n',
        '- **上位の規則**：[ページサイズ](#一覧)\n',
      ),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-002 uses 上位の規則; write the link as 上位の要件')
  })

  it('rejects specification sections out of order', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      SPECIFICATION.replace('## 概要\n\nタスクを扱う。\n', '').concat(
        '\n## 概要\n\nタスクを扱う。\n',
      ),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('section 概要 must come before 操作')
  })

  it('accepts a quality section after operations', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      `${SPECIFICATION}\n## 品質\n\n一覧は 1 秒以内に返す。\n`,
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.output).toContain('ok  specification rules')
    expect(result.code).toBe(0)
  })

  it('rejects an errors section, which TypeSpec and the requirements already state', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/modules/demo/features/task.md',
      `${SPECIFICATION}\n## エラー\n\n共通のエラーはない。\n`,
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('section エラー is not a section of a feature specification')
  })

  it('maps a code slice to a feature slice specification inside a group', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'backend/demo/note/usecases/note.go', 'package usecases\n')
    const result = await runCheck(root, 'specification-rules')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain(
      'backend/demo/note has no feature slice specification under docs/modules/demo/',
    )
    expect(result.output).not.toContain('backend/demo/task has no feature slice specification')
  })
})
