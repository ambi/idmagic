import { describe, expect, it } from 'bun:test'
import { collapseRecord, summaryOf } from './archive.ts'

const path = 'work-items/active/wi-10001-demo.md'

const full = [
  '---',
  'status: in_progress',
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
  '## 要約',
  '',
  '開始を[仕様](../../docs/modules/demo/README.md#開始)へ足した。',
  '`start` を実装した。',
  '',
  '## リスク',
  '',
  'ない。',
  '',
].join('\n')

describe('collapseRecord', () => {
  it('keeps the status, the title, and the summary without links', () => {
    expect(collapseRecord(full, 'completed', path)).toBe(
      [
        '---',
        'status: completed',
        '---',
        '',
        '# デモを開始する',
        '',
        '開始を仕様へ足した。',
        '`start` を実装した。',
        '',
        `全文は \`git log -1 -- ${path}\` が示すコミットの親、またはそのコミットで読む。`,
        '',
      ].join('\n'),
    )
  })

  it('refuses a record without a summary', () => {
    const withoutSummary = full.replace(/## 要約[\s\S]*?(?=## リスク)/, '')
    expect(summaryOf(withoutSummary)).toBeUndefined()
    expect(() => collapseRecord(withoutSummary, 'completed', path)).toThrow('## 要約')
    expect(summaryOf(full.replace(/(## 要約\n\n)[\s\S]*?(?=## リスク)/, '$1'))).toBeUndefined()
  })
})
