import { describe, expect, it } from 'bun:test'
import { verifyWorkItemDependencies, verifyWorkItemIdentifiers } from './work-item-dependencies.ts'

const record = (id: string, depends_on: string[] = []) => ({
  id,
  depends_on,
  path: `work-items/${id}.md`,
  depends_on_line: 6,
})

describe('verifyWorkItemDependencies', () => {
  it('ファイル名の stem が同じ二つの記録を拒否する', () => {
    const findings = verifyWorkItemDependencies([
      record('wi-531-duplicate'),
      { ...record('wi-531-duplicate'), path: 'work-items/done/wi-531-duplicate.md' },
    ])

    expect(findings).toHaveLength(1)
    expect(findings[0]).toMatchObject({
      path: 'work-items/done/wi-531-duplicate.md',
      message: expect.stringContaining("duplicate work item 'wi-531-duplicate'"),
    })
  })

  it('accepts an acyclic graph including completed prerequisites', () => {
    expect(
      verifyWorkItemDependencies([
        record('wi-1-foundation'),
        record('wi-2-feature', ['wi-1-foundation']),
      ]),
    ).toEqual([])
  })

  it('reports an unknown prerequisite at the dependency field', () => {
    const findings = verifyWorkItemDependencies([record('wi-2-feature', ['wi-9-missing'])])
    expect(findings[0]).toMatchObject({
      line: 6,
      message: expect.stringContaining('unknown work item'),
    })
  })

  it('reports self dependency', () => {
    const findings = verifyWorkItemDependencies([record('wi-2-feature', ['wi-2-feature'])])
    expect(findings[0]?.message).toContain('must not depend on itself')
  })

  it('reports indirect cycles once', () => {
    const findings = verifyWorkItemDependencies([
      record('wi-1-a', ['wi-2-b']),
      record('wi-2-b', ['wi-3-c']),
      record('wi-3-c', ['wi-1-a']),
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('wi-1-a -> wi-2-b -> wi-3-c -> wi-1-a')
  })
})

/** 5 桁の枠を `count` 個だけ埋めた記録。空き枠の境目を突くために使う。 */
const fillers = (count: number) =>
  Array.from({ length: count }, (_, index) => record(`wi-${10_000 + index}-filler`))

describe('verifyWorkItemIdentifiers', () => {
  it('題名が違っても識別番号が同じ組を報告する', () => {
    const { findings } = verifyWorkItemIdentifiers([
      record('wi-40318-one'),
      { ...record('wi-40318-two'), path: 'work-items/done/wi-40318-two.md' },
    ])

    expect(findings).toHaveLength(1)
    expect(findings[0]).toMatchObject({
      path: 'work-items/done/wi-40318-two.md',
      message: "work-item identifier: 40318 is shared by 'wi-40318-one' and 'wi-40318-two'",
    })
  })

  it('番号が違えば何も報告しない', () => {
    expect(
      verifyWorkItemIdentifiers([record('wi-40318-one'), record('wi-40319-two')]).findings,
    ).toEqual([])
  })

  it('同じ番号を共有する記録が 3 件あれば、組ごとに報告する', () => {
    const { findings } = verifyWorkItemIdentifiers([
      record('wi-112-alpha'),
      record('wi-112-bravo'),
      record('wi-112-charlie'),
    ])

    expect(findings.map((finding) => finding.message)).toEqual([
      "work-item identifier: 112 is shared by 'wi-112-alpha' and 'wi-112-bravo'",
      "work-item identifier: 112 is shared by 'wi-112-alpha' and 'wi-112-charlie'",
      "work-item identifier: 112 is shared by 'wi-112-bravo' and 'wi-112-charlie'",
    ])
  })

  it('3 桁で起票された新しい記録も、旧来の番号と衝突すれば報告する', () => {
    const { findings } = verifyWorkItemIdentifiers([
      record('wi-634-old'),
      record('wi-634-filed-with-the-retired-rule'),
    ])

    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('634 is shared by')
  })

  it('空き枠が十分にある間は警告を出さない', () => {
    const { warnings } = verifyWorkItemIdentifiers([record('wi-40318-one'), record('wi-634-old')])

    expect(warnings).toEqual([])
  })

  it('空き枠が閾値を下回ると警告を出す', () => {
    const { findings, warnings } = verifyWorkItemIdentifiers(fillers(80_001))

    expect(findings).toEqual([])
    expect(warnings).toEqual([
      'work-item identifier capacity: 9999 of 90000 five-digit numbers remain, below the 10000 warning threshold',
    ])
  })

  it('空き枠がちょうど閾値なら警告を出さない', () => {
    expect(verifyWorkItemIdentifiers(fillers(80_000)).warnings).toEqual([])
  })

  it('5 桁の外にある番号は空き枠を減らさない', () => {
    // 空き枠が閾値ちょうどの状態へ範囲外の記録を足す。範囲で絞らずに数えれば
    // 空き枠が閾値を割り、警告が出てしまう。
    const records = [...fillers(80_000), record('wi-634-old'), record('wi-100000-future')]

    expect(verifyWorkItemIdentifiers(records).warnings).toEqual([])
  })
})
