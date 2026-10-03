import MarkdownIt from 'markdown-it'

type ExampleCase = {
  id: string
  title: string
  given: Record<string, unknown>
  input: Record<string, unknown>
  expected: Record<string, unknown>
}
const markdown = new MarkdownIt({ html: true })
const END = '<!-- /spec:examples -->'

function object(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** 値は独立した期待結果であり、操作や式として解釈しない。 */
export function renderExampleCases(value: unknown): string {
  if (!Array.isArray(value) || value.length === 0)
    throw new Error('examples must be a nonempty array')
  const seen = new Set<string>()
  const cases = value.map((item): ExampleCase => {
    if (
      !object(item) ||
      Object.keys(item).some((key) => !['id', 'title', 'given', 'input', 'expected'].includes(key))
    )
      throw new Error('unknown example field')
    if (typeof item.id !== 'string' || !/^EX-[A-Z0-9-]+-\d+$/.test(item.id) || seen.has(item.id))
      throw new Error('invalid or duplicate example id')
    if (typeof item.title !== 'string' || !item.title.trim() || /[\r\n]/.test(item.title))
      throw new Error('example title must occupy one line')
    if (
      !object(item.given) ||
      !object(item.input) ||
      !object(item.expected) ||
      Object.keys(item.expected).length === 0
    )
      throw new Error('given, input, and nonempty expected must be objects')
    seen.add(item.id)
    return item as ExampleCase
  })
  return cases
    .map((item) =>
      [
        `### Example: ${item.id} ${item.title}`,
        '',
        Object.keys(item.given).length ? '- Given 次の前提を満たす' : '- Given 追加の前提はない',
        table(item.given),
        '- When 次の入力で規則の対象操作を実行する',
        table(item.input),
        '- Then 次の結果になる',
        table(item.expected),
        '',
      ].join('\n'),
    )
    .join('\n')
}

function table(values: Record<string, unknown>): string {
  if (Object.keys(values).length === 0) return ''
  return [
    '',
    '  | 項目 | 値 |',
    '  | --- | --- |',
    ...Object.entries(values).map(
      ([key, value]) => `  | ${cell(key)} | ${cell(JSON.stringify(value))} |`,
    ),
  ].join('\n')
}

function cell(value: string): string {
  return value
    .replaceAll('\\', '\\\\')
    .replaceAll('|', '\\|')
    .replaceAll('\n', '\\n')
    .replaceAll('\r', '\\r')
}

/** コード例を参照と誤認しないよう、Markdown のコメント token だけを入口にする。 */
export function updateExampleBlocks(
  source: string,
  read: (path: string) => string | undefined,
): string {
  const lines = source.split('\n')
  const comments = new Set(
    markdown
      .parse(source, {})
      .filter((token) => token.type === 'html_block')
      .flatMap((token) => {
        const [start, end] = token.map ?? [0, 0]
        return Array.from({ length: end - start }, (_, index) => start + index)
      }),
  )
  const output: string[] = []
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index] ?? ''
    const match = comments.has(index) ? line.match(/^<!-- spec:examples (.+) -->$/) : undefined
    if (!match) {
      if (comments.has(index) && line === END)
        throw new Error('example block has no opening marker')
      if (comments.has(index) && /^<!--\s*\/?spec:examples/.test(line))
        throw new Error('invalid example marker')
      output.push(line)
      continue
    }
    const path = match[1] ?? ''
    if (
      !/^(?:backend|frontend|tools)\/[A-Za-z0-9_./-]+\/testdata\/[A-Za-z0-9_.-]+\.examples\.json$/.test(
        path,
      ) ||
      path.split('/').some((part) => part === '..' || part === '.')
    )
      throw new Error(`invalid example source path: ${path}`)
    const begin = index
    while (++index < lines.length && !(comments.has(index) && lines[index] === END)) {
      if (comments.has(index) && lines[index]?.startsWith('<!-- spec:examples '))
        throw new Error('example blocks cannot be nested')
    }
    if (index === lines.length) throw new Error(`example block at line ${begin + 1} is not closed`)
    const data = read(path)
    if (data === undefined) throw new Error(`example source does not exist: ${path}`)
    output.push(line, '', renderExampleCases(JSON.parse(data)), END)
  }
  return output.join('\n')
}
