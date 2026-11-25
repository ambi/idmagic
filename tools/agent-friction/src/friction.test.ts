import { describe, expect, it } from 'bun:test'
import {
  type FrictionTarget,
  formatFrictionReport,
  type LogRecord,
  parseFrictionTrailer,
  summarizeFriction,
  unreviewedEntries,
} from './friction.ts'

describe('unreviewedEntries', () => {
  const record = (sha: string, frictions: string[], reviewed: string[] = []): LogRecord => ({
    sha,
    date: `2026-11-0${sha.slice(-1)}`,
    frictions,
    reviewed,
  })
  const shas = (entries: ReturnType<typeof unreviewedEntries>) =>
    entries.map((entry) => ('target' in entry ? `${entry.sha}:${entry.detail}` : entry.value))

  /**
   * 判断した対象の古い記録を外さないと、直した後も毎回の集計に繰り返しとして残る。
   * 振り返りのコミット自身の記録は、その振り返りでは判断していないので数える。
   */
  it('振り返りで判断した対象は、そのコミットより前の記録だけを外す', () => {
    const entries = unreviewedEntries([
      record('r4', ['AGENTS.md — 振り返りの後']),
      record('r3', ['AGENTS.md — 振り返りの作業中'], ['AGENTS.md']),
      record('r2', ['AGENTS.md — 判断済み']),
      record('r1', ['AGENTS.md — 判断済み']),
    ])
    expect(shas(entries)).toEqual(['r3:振り返りの作業中', 'r4:振り返りの後'])
  })

  /** 判断しなかった 1 件だけの記録も外すと、次の記録と合わさって 2 件になる機会を失う。 */
  it('振り返りで判断しなかった対象は全履歴から数える', () => {
    const entries = unreviewedEntries([
      record('r3', ['docs/README.md — 二回目']),
      record('r2', [], ['AGENTS.md']),
      record('r1', ['docs/README.md — 一回目']),
    ])
    expect(shas(entries)).toEqual(['r1:一回目', 'r3:二回目'])
  })

  it('アンカーまで一致する対象だけを判断済みとする', () => {
    const entries = unreviewedEntries([
      record('r2', [], ['docs/a.md#x']),
      record('r1', ['docs/a.md#x — 判断済み', 'docs/a.md#y — 別の節']),
    ])
    expect(shas(entries)).toEqual(['r1:別の節'])
  })

  it('対象として読めない目印を形式外として残す', () => {
    expect(shas(unreviewedEntries([record('r1', [], ['mise run'])]))).toEqual([
      'Agent-Friction-Reviewed: mise run',
    ])
  })
})

describe('parseFrictionTrailer', () => {
  it('アンカー付きのパスと起きたことを読む', () => {
    expect(
      parseFrictionTrailer(
        'a1',
        '2026-11-20',
        'docs/development/testing.md#テストダブル — 共有契約の節を読まずに時刻の分解能を誤った',
      ),
    ).toEqual({
      sha: 'a1',
      date: '2026-11-20',
      target: { kind: 'path', path: 'docs/development/testing.md', anchor: 'テストダブル' },
      detail: '共有契約の節を読まずに時刻の分解能を誤った',
    })
  })

  it('mise run で始まる対象をタスクとして読む', () => {
    expect(
      parseFrictionTrailer('a2', '2026-11-21', 'mise run test-tools — 初回の実行が失敗した'),
    ).toEqual({
      sha: 'a2',
      date: '2026-11-21',
      target: { kind: 'task', task: 'test-tools' },
      detail: '初回の実行が失敗した',
    })
  })

  /** 区切りのない値は、どの対象として数えるかを決められない。捨てると書いた記録が黙って消える。 */
  it('区切りのない値と、対象か起きたことが空の値を形式外とする', () => {
    for (const value of [
      'docs/README.md が古い',
      ' — 対象がない',
      'docs/README.md — ',
      'mise run  — タスク名がない',
    ]) {
      expect(parseFrictionTrailer('a3', '2026-11-22', value)).toEqual({
        sha: 'a3',
        date: '2026-11-22',
        value,
      })
    }
  })
})

describe('summarizeFriction', () => {
  const entry = (sha: string, value: string) =>
    parseFrictionTrailer(sha, '2026-11-2' + sha.slice(-1), value)
  const exists = (target: FrictionTarget) =>
    target.kind === 'path' ? target.path !== 'docs/removed.md' : target.task !== 'removed-task'

  const report = summarizeFriction(
    [
      entry('c1', 'docs/development/testing.md#テストダブル — 一回目'),
      entry('c2', 'docs/development/testing.md#テストダブル — 二回目'),
      entry('c3', 'docs/development/testing.md — アンカーのない別の対象'),
      entry('c4', 'docs/removed.md — 消えた文書'),
      entry('c5', 'mise run removed-task — 消えたタスク'),
      entry('c6', '区切りのない値'),
    ],
    exists,
  )

  it('アンカーまで一致する対象を一つとして数え、2 件以上を繰り返しに分ける', () => {
    expect(report.recurring.map((summary) => [summary.label, summary.entries.length])).toEqual([
      ['docs/development/testing.md#テストダブル', 2],
    ])
    expect(report.once.map((summary) => summary.label)).toEqual(['docs/development/testing.md'])
  })

  /** 対象が消えていれば、繰り返しを数えるより先に、記録が指す先が今どこにあるかを決める必要がある。 */
  it('現在のツリーにない対象は件数によらず別に分ける', () => {
    expect(report.missing.map((summary) => summary.label)).toEqual([
      'docs/removed.md',
      'mise run removed-task',
    ])
  })

  it('形式外の値を捨てずに残す', () => {
    expect(report.malformed).toEqual([{ sha: 'c6', date: '2026-11-26', value: '区切りのない値' }])
  })
})

describe('formatFrictionReport', () => {
  it('区分ごとに対象、件数、日付、コミット、起きたことを示す', () => {
    const report = summarizeFriction(
      [
        parseFrictionTrailer('1111111aaaa', '2026-11-20', 'AGENTS.md — 一回目'),
        parseFrictionTrailer('2222222bbbb', '2026-11-21', 'AGENTS.md — 二回目'),
      ],
      () => true,
    )
    expect(formatFrictionReport(report)).toBe(
      [
        '## Recurring (2 or more)',
        '',
        '- AGENTS.md: 2',
        '  - 2026-11-20 1111111 一回目',
        '  - 2026-11-21 2222222 二回目',
        '',
        '## Once',
        '',
        'none',
        '',
        '## Missing from the current tree',
        '',
        'none',
        '',
        '## Malformed',
        '',
        'none',
        '',
      ].join('\n'),
    )
  })
})
