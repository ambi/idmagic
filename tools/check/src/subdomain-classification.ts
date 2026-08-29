/**
 * `docs/README.md` の Context 索引表が、全 Bounded Context をサブドメインの
 * 区分つきで 1 行ずつ持つことを確かめる。
 *
 * 分類は、それに依存するものが無ければ飾りになる。索引表は新しい Context が
 * 増えるたびに書き足される場所であり、列を足しただけでは次の 1 件が空欄のまま
 * 通る。ここが拒否を引き受けるのは、分類の欠落を書いた人の手元で止めるためで
 * ある。
 *
 * 区分そのものが妥当かどうかは検査しない。Core と Supporting の境目は判断で
 * あって、その理由は各 Context の `decisions.md` にある。ここが見るのは、
 * 判断が書かれていない行が無いことだけである。
 */

/** 索引表が引ける区分。`docs/design-rules.md` が何を左右するかを定める。 */
const SUBDOMAINS = ['Core', 'Supporting', 'Generic'] as const

/** Context 索引表を選ぶヘッダー行。見出しではなく列の並びで表を特定する。 */
const INDEX_HEADER = '| Specification context | Subdomain | Go package | Responsibility |'

/** 区分の列を持たない、変更前のヘッダー行。 */
const UNCLASSIFIED_HEADER = '| Specification context | Go package | Responsibility |'

export interface SubdomainFinding {
  line: number
  message: string
}

/** 行を `|` で分けたセル。前後の空セルは表の縁なので落とす。 */
function cells(row: string): string[] {
  const parts = row.split('|').map((cell) => cell.trim())
  if (parts[0] === '') parts.shift()
  if (parts.at(-1) === '') parts.pop()
  return parts
}

/** `[Name](contexts/<dir>/README.md)` が指す Context ディレクトリ名。 */
function contextDirectory(cell: string): string | undefined {
  return cell.match(/\(contexts\/([^/)]+)\/README\.md\)/)?.[1]
}

/**
 * `source` は `docs/README.md` の本文、`contextDirectories` は
 * `docs/contexts/` の直下にあるディレクトリ名。どちらも引数で入るので、この
 * 関数はファイルシステムも作業ディレクトリも読まない。
 */
export function verifySubdomainClassification(
  source: string,
  contextDirectories: string[],
): SubdomainFinding[] {
  // Context が 1 つも無い作業ツリーには索引すべきものが無い。ここで表を要求すると、
  // 中身より先に体裁を書かせることになる。
  if (contextDirectories.length === 0) return []

  const lines = source.split(/\r?\n/)
  const headerIndex = lines.findIndex((line) => line.trim() === INDEX_HEADER)
  if (headerIndex < 0) {
    const unclassified = lines.findIndex((line) => line.trim() === UNCLASSIFIED_HEADER)
    return [
      {
        line: unclassified >= 0 ? unclassified + 1 : 1,
        message:
          unclassified >= 0
            ? `the context index table declares no Subdomain column; its header must read ${INDEX_HEADER}`
            : `no context index table found; its header must read ${INDEX_HEADER}`,
      },
    ]
  }

  const findings: SubdomainFinding[] = []
  const listed = new Map<string, number>()
  const known = new Set(contextDirectories)
  const allowed = new Set<string>(SUBDOMAINS)

  // ヘッダーの次は区切り行なので、その 1 つ下から表が終わるまでを読む。
  for (let index = headerIndex + 2; index < lines.length; index += 1) {
    const row = lines[index]!.trim()
    if (!row.startsWith('|')) break
    const columns = cells(row)
    const directory = contextDirectory(columns[0] ?? '')
    if (!directory) continue
    const line = index + 1

    const first = listed.get(directory)
    if (first !== undefined) {
      findings.push({
        line,
        message: `${directory} is listed twice; it also appears on line ${first}`,
      })
      continue
    }
    listed.set(directory, line)

    if (!known.has(directory)) {
      findings.push({ line, message: `docs/contexts/${directory}/ does not exist` })
      continue
    }

    const subdomain = columns[1] ?? ''
    if (!allowed.has(subdomain)) {
      findings.push({
        line,
        message: subdomain
          ? `${directory} is classified as ${subdomain}; the Subdomain column holds one of ${SUBDOMAINS.join(', ')}`
          : `${directory} declares no subdomain; the Subdomain column holds one of ${SUBDOMAINS.join(', ')}`,
      })
    }
  }

  for (const directory of contextDirectories) {
    if (listed.has(directory)) continue
    findings.push({
      line: headerIndex + 1,
      message: `docs/contexts/${directory}/ is not listed in the context index table`,
    })
  }

  return findings
}
