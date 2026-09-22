import { describe, expect, it } from 'bun:test'
import { isCurrentStateDocument, verifyNoWorkItemLinks } from './docs-work-item-links.ts'

describe('現在状態の文書からの work item 参照', () => {
  it('work item の番号と、work-items へのリンクを列の順に指摘する', () => {
    const findings = verifyNoWorkItemLinks([
      {
        file: 'docs/design/performance/README.md',
        source: '# 性能設計\n\n検証は [wi-282](../../../work-items/wi-282-staging.md) が扱う。\n',
      },
    ])

    expect(findings).toEqual([
      { file: 'docs/design/performance/README.md', line: 3, column: 6, reference: 'wi-282' },
      { file: 'docs/design/performance/README.md', line: 3, column: 23, reference: 'work-items/' },
      { file: 'docs/design/performance/README.md', line: 3, column: 34, reference: 'wi-282' },
    ])
  })

  // ディレクトリ構成の説明はリンクではないので、変更の記録への依存にならない。
  it('リンクではない work-items/ の記述は通す', () => {
    expect(
      verifyNoWorkItemLinks([
        { file: 'docs/README.md', source: '変更の記録は `work-items/` が持つ。\n' },
      ]),
    ).toEqual([])
  })

  it('語の一部としての wi- は指摘しない', () => {
    expect(
      verifyNoWorkItemLinks([
        {
          file: 'docs/design/data/database.md',
          source: 'kiwi-42、awi-7、Wi-Fi、wiki は work item ではない。\n',
        },
      ]),
    ).toEqual([])
  })

  // 変更単位で書くリリースノートと、手順の例を持つ開発文書は対象にしない。
  it('現在状態を書く文書だけを対象にする', () => {
    expect(isCurrentStateDocument('docs/design/infrastructure/network.md')).toBe(true)
    expect(isCurrentStateDocument('docs/design/architecture/deployment.md')).toBe(true)
    expect(isCurrentStateDocument('docs/requirements/quality.md')).toBe(true)
    expect(isCurrentStateDocument('docs/domain/oauth2/decisions.md')).toBe(true)
    expect(isCurrentStateDocument('docs/runbooks/async-jobs.md')).toBe(true)
    expect(isCurrentStateDocument('docs/README.md')).toBe(true)
    expect(isCurrentStateDocument('docs/releases/changes/wi-532.md')).toBe(false)
    expect(isCurrentStateDocument('docs/development/local-development.md')).toBe(false)
    expect(isCurrentStateDocument('docs/design/infrastructure/network.yaml')).toBe(false)
    expect(isCurrentStateDocument('work-items/wi-584-example.md')).toBe(false)
  })
})
