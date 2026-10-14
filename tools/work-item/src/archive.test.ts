import { describe, expect, it } from 'bun:test'
import { collapseRecord } from './archive.ts'

const full = [
  '---',
  'status: completed',
  'authors: [tn]',
  'affected_spec:',
  '  - { path: docs/modules/demo/README.md, requirement: REQ-DEMO-001 }',
  '---',
  '',
  '# デモを開始する',
  '',
  '## 動機',
  '',
  '開始できない。',
  '',
  '## 完了',
  '- **完了日**: 2026-01-01',
  '- **要約**:',
  '  開始を[仕様](../../docs/modules/demo/README.md#開始)へ足した。',
  '  `start` を実装した。',
  '- **検証結果**:',
  '  - 成功。',
  '',
].join('\n')

describe('collapseRecord', () => {
  it('keeps the status, the revision, the title, and the completion summary without links', () => {
    expect(collapseRecord(full, 'abc1234')).toBe(
      [
        '---',
        'status: completed',
        'archived_from: abc1234',
        '---',
        '',
        '# デモを開始する',
        '',
        '開始を仕様へ足した。',
        '`start` を実装した。',
        '',
        '全文は `git show abc1234:<このファイルのパス>` で読む。',
        '',
      ].join('\n'),
    )
  })

  it('falls back to the English summary field and then to the motivation', () => {
    const english = full.replace('**要約**', '**Summary**')
    expect(collapseRecord(english, 'abc1234')).toContain('`start` を実装した。')
    const noSummary = full.replace(/- \*\*要約\*\*:[\s\S]*?(?=- \*\*検証結果)/, '')
    expect(collapseRecord(noSummary, 'abc1234')).toContain('\n開始できない。\n')
  })

  it('leaves an already collapsed record unchanged', () => {
    const collapsed = collapseRecord(full, 'abc1234')
    expect(collapseRecord(collapsed, 'def5678')).toBe(collapsed)
  })
})
