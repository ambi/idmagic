/**
 * 正本文書のファイル集合が閉じていることを確かめる。
 *
 * 集める側（`discoverSpecificationDocuments`）は許可名だけを拾う絞り込みであり、それ自体
 * は正しい。差分を取るだけの操作が未登録ファイルを理由に落ちてはならないからである。
 * その結果として、名前を打ち間違えた正本文書は書いた人からは存在して見え、検査からは
 * 存在しないという状態が生まれる。ここが拒否を引き受けるのは、集める責務と拒否する
 * 責務を分けたままにするためである。
 *
 * 主な害が打ち間違いの見逃しである以上、失敗は「許可されていない」で終わってはならない。
 * 近い許可名を示して、書いた人が何を間違えたかに到達させる。
 */

import { CONTEXT_DOCUMENTS, ROOT_DOCUMENTS } from './specification-doc.ts'

/** 一段のディレクトリと、その直下にあるファイル名。 */
export interface DirectoryListing {
  directory: string
  files: string[]
}

export interface Finding {
  path: string
  message: string
}

/** 打ち間違いとみなす編集距離の上限。これを超える名前には候補を示さない。 */
const SUGGESTION_DISTANCE = 2

/** その段が許す名前。`docs/` 直下と Context 直下は別の集合を持つ。 */
function allowedNames(directory: string): readonly string[] {
  return directory === 'docs' ? ROOT_DOCUMENTS : CONTEXT_DOCUMENTS
}

function isMarkdown(name: string): boolean {
  return name.toLowerCase().endsWith('.md')
}

/** Levenshtein 距離。名前の長さは高々数十文字なので素朴な実装で足りる。 */
function distance(a: string, b: string): number {
  let previous = Array.from({ length: b.length + 1 }, (_, index) => index)
  for (let i = 1; i <= a.length; i += 1) {
    const current = [i]
    for (let j = 1; j <= b.length; j += 1) {
      const substitution = previous[j - 1]! + (a[i - 1] === b[j - 1] ? 0 : 1)
      current.push(Math.min(substitution, previous[j]! + 1, current[j - 1]! + 1))
    }
    previous = current
  }
  return previous[b.length]!
}

/**
 * 打ち間違いとして最も近い許可名。近いものが無ければ返さない。
 *
 * 距離は両辺を小文字にして測る。そのため `glossary.MD` のように大文字小文字だけが
 * 違う名前は距離 0 になり、必ず候補として返る。拡張子の大文字は打ち間違いの中でも
 * 特に見つけにくい部類なので、この扱いが要る。
 */
function nearestName(name: string, allowed: readonly string[]): string | undefined {
  const lower = name.toLowerCase()
  let nearest: string | undefined
  let nearestDistance = SUGGESTION_DISTANCE + 1
  for (const candidate of allowed) {
    const measured = distance(lower, candidate.toLowerCase())
    if (measured < nearestDistance) {
      nearest = candidate
      nearestDistance = measured
    }
  }
  return nearest
}

/**
 * 許可リストに無い Markdown を所見として返す。Markdown 以外は対象にしない。
 * 図表や添付をどう扱うかは、この検査が決めることではない。
 */
export function verifyCanonicalDocumentSet(listings: DirectoryListing[]): Finding[] {
  const findings: Finding[] = []
  for (const listing of listings) {
    const allowed = allowedNames(listing.directory)
    for (const name of listing.files) {
      if (!isMarkdown(name) || allowed.includes(name)) continue
      const nearest = nearestName(name, allowed)
      findings.push({
        path: `${listing.directory}/${name}`,
        message: nearest
          ? `not a canonical document; did you mean ${nearest}?`
          : `not a canonical document; ${listing.directory}/ holds only ${allowed.join(', ')}`,
      })
    }
  }
  return findings
}
