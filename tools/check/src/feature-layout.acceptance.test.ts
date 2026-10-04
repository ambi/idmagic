/**
 * 機能仕様と内部設計を軸にした形式の Context を、検査が受け入れ、崩れを拒否することを確かめる。
 *
 * Context の印は `design/README.md` であり、この作業ツリーはそれを持つ Context `demo` だけで組む。
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

const TOPICS = [
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
]

/** 話題の索引。アーキテクチャだけを記述し、ほかの話題は該当しない理由を書く。 */
function topicIndex(title: string, architecture: string): string {
  return [
    `# ${title}`,
    '',
    '| 話題 | 記述した場所 |',
    '| --- | --- |',
    ...TOPICS.map((topic) =>
      topic === 'アーキテクチャ'
        ? `| ${topic} | ${architecture} |`
        : `| ${topic} | 該当なし：Demo は小さく、固有の設計がない |`,
    ),
    '',
  ].join('\n')
}

/** Context を一つだけ持ち、どの検査にも通る作業ツリー。 */
async function featureLayoutWorkspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-feature-layout-test-'))
  cleanup.push(root)
  const files: Record<string, string> = {
    'docs/README.md': '# Specification\n',
    'docs/design/architecture/logical.md': [
      '# 論理アーキテクチャ',
      '',
      '| 仕様上の Context | Subdomain | Go パッケージ | 責務 |',
      '| --- | --- | --- | --- |',
      '| [Demo](../../domain/demo/README.md) | Core | `demo` | Demo. |',
      '',
    ].join('\n'),
    'docs/domain/demo/README.md': '# Demo\n',
    'docs/domain/demo/glossary.md': '# Demo の用語集\n',
    'docs/domain/demo/design/README.md': topicIndex(
      'Demo の設計',
      '[アーキテクチャ](architecture.md)',
    ),
    'docs/domain/demo/design/architecture.md': '# Demo のアーキテクチャ\n',
    'docs/domain/demo/design/decisions.md': DECISIONS,
    'docs/domain/demo/design/audit.md': '# 監査\n',
    'docs/domain/demo/work/README.md': '# 作業\n',
    'docs/domain/demo/work/task/README.md': SPECIFICATION,
    'docs/domain/demo/work/task/design.md': `${topicIndex('タスクの設計', '[アーキテクチャ](#アーキテクチャ)')}\n## アーキテクチャ\n\nタスクを保存する。\n`,
    'docs/domain/demo/work/task/acceptance.feature.md': EXAMPLES,
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

  it('rejects the per-kind files of the legacy layout inside a feature slice', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/domain/demo/work/task/decisions.md', '# 判断\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/domain/demo/work/task/decisions.md')
  })

  it('accepts a requirement that has no example in the appendix', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/work/task/README.md',
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
      'docs/domain/demo/work/task/acceptance.feature.md',
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
      'docs/domain/demo/work/task/acceptance.feature.md',
      EXAMPLES.replace('開いたタスクだけを一覧する', 'タスクを一覧する'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-002 title differs from the feature specification')
  })

  it('rejects a rule declared in a group of features', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/work/README.md',
      '# 作業\n\n#### REQ-DEMO-004 作業を数える\n\n- 作業を数える。\n',
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('REQ-DEMO-004 must be declared in a feature slice')
  })

  it('rejects a decision that leaves out a part of the decision record', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/design/decisions.md',
      DECISIONS.replace('### 検討した代替案', '### 比較'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('タスクを物理削除しない must have the section 検討した代替案')
  })

  it('accepts the quality requirements allocated to a context', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/domain/demo/quality.md', '# Demo の品質要件\n')
    const result = await runCheck(root, 'documents')
    expect(result.output).not.toContain('docs/domain/demo/quality.md')
    expect(result.code).toBe(0)
  })

  it('rejects a design entry point without a topic index', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/domain/demo/design/README.md', '# Demo の設計\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('must have a topic index')
  })

  it('accepts a feature design without a topic index', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/work/task/design.md',
      '# タスクの設計\n\n## 信頼性\n\n削除が途中で失敗したら、削除を再実行する。\n',
    )
    const result = await runCheck(root, 'documents')
    expect(result.output).not.toContain('docs/domain/demo/work/task/design.md')
    expect(result.code).toBe(0)
  })

  it('rejects a topic index that leaves out a topic or names an unknown one', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/design/README.md',
      topicIndex('Demo の設計', '[アーキテクチャ](architecture.md)')
        .replace(/\| 性能 \|.*\n/, '')
        .replace('| リスク |', '| 運用 |'),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('topic index must list 性能')
    expect(result.output).toContain('topic index names an unknown topic 運用')
  })

  it('rejects a topic that neither links to its design nor says why it does not apply', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/design/README.md',
      topicIndex('Demo の設計', 'architecture.md に書いた').replace(
        '| データ | 該当なし：Demo は小さく、固有の設計がない |',
        '| データ | 該当なし： |',
      ),
    )
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('topic アーキテクチャ must link to its design')
    expect(result.output).toContain('topic データ must link to its design')
  })

  it('applies the topic index to the system design', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/design/README.md', '# 設計文書\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/design/README.md')
    expect(result.output).toContain('must have a topic index')
  })

  it('requires every context to have design/README.md', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/domain/other/README.md', '# Other\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('other has no design/README.md')
  })

  it('rejects the per-kind files of the former layout directly under a context', async () => {
    const root = await featureLayoutWorkspace()
    await write(root, 'docs/domain/demo/scenarios.feature.md', '# Feature: Demo\n')
    await write(root, 'docs/domain/demo/states.md', '# Demo の状態\n')
    const result = await runCheck(root, 'documents')
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/domain/demo/scenarios.feature.md')
    expect(result.output).toContain('docs/domain/demo/states.md')
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
      'docs/domain/demo/work/task/README.md',
      SPECIFICATION.replace('- **担保手段**：`Task.Open`\n', ''),
    )
    const result = await runCheck(root, 'specification-rules')
    expect(result.output).toContain('ok  specification rules')
    expect(result.code).toBe(0)
  })

  it('rejects the legacy reason field', async () => {
    const root = await featureLayoutWorkspace()
    await write(
      root,
      'docs/domain/demo/work/task/README.md',
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
      'docs/domain/demo/work/task/README.md',
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
      'docs/domain/demo/work/task/README.md',
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
      'docs/domain/demo/work/task/README.md',
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
      'docs/domain/demo/work/task/README.md',
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
      'backend/demo/note has no feature slice specification under docs/domain/demo/',
    )
    expect(result.output).not.toContain('backend/demo/task has no feature slice specification')
  })
})
