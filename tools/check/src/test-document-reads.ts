import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'

/**
 * リポジトリの根からの `docs/`、またはパッケージから `../` でさかのぼった先の `docs/`。
 * パッケージの中の `testdata/docs/`、URL の途中の `/docs/`、名前としての `docs` は文書のパスではない。
 */
const DOCUMENT_PATH = /^(?:\.\/)?(?:\.\.\/)*docs\//

/**
 * Go のソースの文字列リテラルのうち、文書を指すパスを返す。コメントの中の `docs/` は、
 * テストが固定する仕様の出典を示す書き方なので数えない。
 */
export function documentPathLiterals(source: string): Array<{ line: number; value: string }> {
  const found: Array<{ line: number; value: string }> = []
  let line = 1
  let index = 0
  const take = (end: number) => {
    const text = source.slice(index, end)
    line += text.split('\n').length - 1
    index = end
    return text
  }
  while (index < source.length) {
    const char = source[index]
    const next = source[index + 1]
    if (char === '/' && next === '/') {
      const end = source.indexOf('\n', index)
      take(end < 0 ? source.length : end)
    } else if (char === '/' && next === '*') {
      const end = source.indexOf('*/', index + 2)
      take(end < 0 ? source.length : end + 2)
    } else if (char === '"' || char === "'" || char === '`') {
      const start = line
      let end = index + 1
      while (end < source.length && source[end] !== char) {
        if (char !== '`' && source[end] === '\\') end += 1
        if (char !== '`' && source[end] === '\n') break
        end += 1
      }
      const literal = take(Math.min(end + 1, source.length))
      const value = literal.slice(1, -1)
      if (char !== "'" && DOCUMENT_PATH.test(value)) found.push({ line: start, value })
    } else {
      take(index + 1)
    }
  }
  return found
}

/**
 * Go のテストが文書の Markdown を入力にすると、文書だけの変更がテストの結果を変え、
 * 変更の影響からテストを選べなくなる。テストの入力はパッケージの `testdata` に置く。
 */
export async function checkTestDocumentReads(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const lines: string[] = []
  const tests = (await snapshot.files('backend')).filter((path) => path.endsWith('_test.go'))
  for (const path of tests) {
    for (const literal of documentPathLiterals(await snapshot.read(path)))
      lines.push(
        `${path}:${literal.line}: test names documentation ${JSON.stringify(literal.value)}; ` +
          "move the input to the package's testdata",
      )
  }
  return lines.length
    ? { ok: false, lines }
    : { ok: true, lines: [`ok  ${tests.length} Go test file(s) read no documentation`] }
}
