import { afterEach, describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkWorkItems, loadWorkItems } from './check-work-items.ts'
import { validateMarkdownRecord } from './work-item-markdown.ts'

const temporaryDirectories: string[] = []

async function unpublishedWorkspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-work-items-'))
  temporaryDirectories.push(root)
  await mkdir(join(root, 'spec'), { recursive: true })
  await writeFile(join(root, 'spec', 'release-state.json'), '{"phase":"unpublished"}')
  return root
}

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })))
})

describe('loadWorkItems', () => {
  it('一覧、本文、解析結果を対象ごとに一度だけ作る', async () => {
    const root = await unpublishedWorkspace()
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    await mkdir(join(root, 'work-items', 'active'), { recursive: true })
    const source =
      '---\nstatus: pending\nauthors: [tn]\nrisk: low\ncreated_at: 2026-09-12\n---\n\n# Test\n'
    await writeFile(join(root, 'work-items', 'active', 'wi-901-one.md'), source)
    await writeFile(join(root, 'work-items', 'done', 'wi-902-two.md'), source)

    const base = createWorkspaceSnapshot(root)
    const listCount = new Map<string, number>()
    const readCount = new Map<string, number>()
    const snapshot = {
      ...base,
      list: async (directory = '') => {
        listCount.set(directory, (listCount.get(directory) ?? 0) + 1)
        return base.list(directory)
      },
      read: async (path: string) => {
        readCount.set(path, (readCount.get(path) ?? 0) + 1)
        return base.read(path)
      },
    }
    let parseCount = 0
    const records = await loadWorkItems(snapshot, (...args) => {
      parseCount++
      return validateMarkdownRecord(...args)
    })

    expect(records.map((record) => record.id).sort()).toEqual(['wi-901-one', 'wi-902-two'])
    expect(listCount).toEqual(
      new Map([
        ['work-items/active', 1],
        ['work-items/done', 1],
      ]),
    )
    expect(readCount).toEqual(
      new Map([
        ['work-items/active/wi-901-one.md', 1],
        ['work-items/done/wi-902-two.md', 1],
      ]),
    )
    expect(parseCount).toBe(2)
  })
})

/** スキーマを満たす最小の記録。番号の衝突だけを唯一の所見として残すために使う。 */
const minimalRecord = (title: string, status = 'pending') =>
  [
    '---',
    `status: ${status}`,
    'authors: [tn]',
    'risk: low',
    'created_at: 2026-09-21',
    'priority: p2',
    'depends_on: []',
    'change_kind: tooling',
    'spec_impact: { kind: none, reason: "検査の fixture であり、規範要素を変えない。" }',
    '---',
    '',
    `# ${title}`,
    '',
    '## 動機',
    '',
    '検査の fixture である。',
    '',
    '## 対象範囲',
    '',
    '- 何も変えない。',
    '',
    '## 対象外',
    '',
    '- 何も変えない。',
    '',
    '## 検証',
    '',
    '- `mise run check-work-items`',
    '',
    '## リスク',
    '',
    'なし。',
    '',
    ...(status === 'cancelled'
      ? ['## 完了', '', '- **完了日**: 2026-09-21', '- **要約**: 中止した。', '']
      : []),
  ].join('\n')

describe('checkWorkItems', () => {
  it('同じ機能追加の記録を未公開では通し、公開後は告知不足で拒否する', async () => {
    const root = await unpublishedWorkspace()
    await mkdir(join(root, 'work-items', 'active'), { recursive: true })
    await mkdir(join(root, 'docs'), { recursive: true })
    await mkdir(join(root, 'spec', 'generated', 'openapi'), { recursive: true })
    await writeFile(join(root, 'docs', 'feature.md'), '### REQ-SYSTEM-001: 機能の規則\n')
    await writeFile(join(root, 'spec', 'idmagic.openapi.baseline.json'), '{"paths":{}}')
    await writeFile(join(root, 'spec', 'generated', 'openapi', 'current.json'), '{"paths":{}}')
    const metadata = [
      'evidence_policy: risk-based-v4',
      'initial_context: { specification: [], typespec: [], source: [], tests: [], stop_before_reading: [] }',
      'documentation_impact: { level: none, reason: "初回公開前であり、利用者向けの告知差分はない。", references: [] }',
      'affected_spec: [{ path: docs/feature.md, requirement: REQ-SYSTEM-001 }]',
      'primary_use_cases:',
      '  - id: feature',
      '    requirement: REQ-SYSTEM-001',
      '    observable_result: 機能の結果を観測できる。',
      '    boundary: unit',
      '    test: { path: backend/feature_test.go, name: TestFeature, task: test-go-race }',
      '    fault_model: 機能を実行しない。',
    ].join('\n')
    const source = minimalRecord('機能追加', 'in_progress')
      .replace('change_kind: tooling', 'change_kind: feature')
      .replace('---\n\n#', `${metadata}\n---\n\n#`)
    await writeFile(join(root, 'work-items', 'active', 'wi-40318-feature.md'), source)

    const before = await checkWorkItems(createWorkspaceSnapshot(root))
    expect(before).toEqual({ ok: true, lines: ['ok  1 work-item dependency record(s)'] })

    await writeFile(join(root, 'spec', 'release-state.json'), '{"phase":"published"}')
    const after = await checkWorkItems(createWorkspaceSnapshot(root))
    expect(after.ok).toBe(false)
    expect(after.lines).toEqual([
      'work-items/active/wi-40318-feature.md: documentation_impact none is weaker than inferred release_note',
    ])
  })

  it('公開状態が欠落している workspace を告知免除にしない', async () => {
    const root = await unpublishedWorkspace()
    await mkdir(join(root, 'work-items'), { recursive: true })
    await rm(join(root, 'spec', 'release-state.json'))
    await expect(checkWorkItems(createWorkspaceSnapshot(root))).rejects.toThrow(
      'release-state.json',
    )
  })

  it('題名が違っても識別番号が同じ二つの記録を落とす', async () => {
    const root = await unpublishedWorkspace()
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    await mkdir(join(root, 'work-items', 'active'), { recursive: true })
    await writeFile(join(root, 'work-items', 'active', 'wi-40318-one.md'), minimalRecord('One'))
    await writeFile(
      join(root, 'work-items', 'done', 'wi-40318-two.md'),
      minimalRecord('Two', 'cancelled'),
    )

    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))

    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain(
      "work-item identifier: 40318 is shared by 'wi-40318-one' and 'wi-40318-two'",
    )
  })

  it('番号が重ならない記録だけの workspace を通す', async () => {
    const root = await unpublishedWorkspace()
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    await mkdir(join(root, 'work-items', 'active'), { recursive: true })
    await writeFile(join(root, 'work-items', 'active', 'wi-40318-one.md'), minimalRecord('One'))
    await writeFile(
      join(root, 'work-items', 'done', 'wi-40319-two.md'),
      minimalRecord('Two', 'cancelled'),
    )

    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))

    expect(outcome).toEqual({ ok: true, lines: ['ok  2 work-item dependency record(s)'] })
  })

  it('旧配置と status に反するディレクトリを拒否する', async () => {
    const root = await unpublishedWorkspace()
    await mkdir(join(root, 'work-items', 'active'), { recursive: true })
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    await writeFile(join(root, 'work-items', 'wi-10001-old.md'), minimalRecord('旧配置'))
    await writeFile(
      join(root, 'work-items', 'active', 'wi-10002-closed.md'),
      minimalRecord('中止', 'cancelled'),
    )
    await writeFile(join(root, 'work-items', 'done', 'wi-10003-open.md'), minimalRecord('未完了'))

    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))

    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain('wi-10001-old.md: legacy work item location')
    expect(outcome.lines.join('\n')).toContain(
      'wi-10002-closed.md: status cancelled belongs in work-items/done',
    )
    expect(outcome.lines.join('\n')).toContain(
      'wi-10003-open.md: status pending belongs in work-items/active',
    )
  })
})
