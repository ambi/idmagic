/**
 * 新しい作業項目へ渡す識別番号を選ぶ。
 *
 * 連番は、番号を配る主体が一つであることを前提にする。複数のメンバーがそれぞれの
 * ローカルで起票し、同じ人が並列のワークツリーで起票する運用ではその前提が崩れる。
 * 最大値 + 1 を数える手順は、まだ push されていない起票を見られないからである。
 * 代わりに未使用の値を無作為に選ぶ。採番規則は `WORK_ITEM_FORMAT.md` が定める。
 *
 * 乱数と走査はここに持たない。`random` は引数で受け取り、ディレクトリの読み取りは
 * `main.ts` が行う。同じ入力なら同じ値を返すので、使用済みの値を返さないという
 * 主張を、縮めた範囲で検査できる。
 */

import {
  IDENTIFIER_RANGE,
  type IdentifierRange,
  workItemNumber,
} from '../../check/src/work-item-dependencies.ts'

/** ファイル名の一覧から、桁数を問わず使用済みの識別番号を集める。 */
export function usedWorkItemNumbers(fileNames: Iterable<string>): Set<number> {
  const used = new Set<number>()
  for (const fileName of fileNames) {
    const number = workItemNumber(fileName.replace(/\.md$/, ''))
    if (number !== undefined) used.add(number)
  }
  return used
}

/**
 * 範囲の中で未使用の値を一様に選ぶ。
 *
 * 棄却法を使わないのは、空き枠が少ないほど試行が延び、埋まった範囲では終わらない
 * からである。空き枠を数えてから対応する枠を返せば、試行回数は範囲の幅で頭打ちになる。
 */
export function pickWorkItemNumber(
  used: ReadonlySet<number>,
  random: () => number,
  range: IdentifierRange = IDENTIFIER_RANGE,
): number {
  let free = 0
  for (let value = range.min; value <= range.max; value++) if (!used.has(value)) free++
  if (free === 0) {
    throw new Error(`no unused work item number remains in ${range.min}-${range.max}`)
  }

  // `random()` が 1 を返す実装に備えて端を丸める。範囲の外を返すより、端の値が
  // わずかに出やすいほうが害が小さい。
  let offset = Math.min(free - 1, Math.max(0, Math.floor(random() * free)))
  for (let value = range.min; value <= range.max; value++) {
    if (used.has(value)) continue
    if (offset === 0) return value
    offset--
  }
  throw new Error(`no unused work item number remains in ${range.min}-${range.max}`)
}
