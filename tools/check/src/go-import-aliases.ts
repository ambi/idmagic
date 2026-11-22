/** 特性化テストの比較用に、曖昧でない import の修飾参照だけを参照先へ置き換える。 */
type Token = { text: string; start: number; end: number }

/** コメントとリテラルの中を識別子として読まないための字句分割。本文の整形は変えない。 */
function tokensOf(source: string): Token[] | undefined {
  const tokens: Token[] = []
  let index = 0
  while (index < source.length) {
    const start = index
    const char = source[index]!
    if (/\s/.test(char)) {
      index++
      continue
    }
    if (source.startsWith('//', index)) {
      const end = source.indexOf('\n', index + 2)
      index = end < 0 ? source.length : end
      continue
    }
    if (source.startsWith('/*', index)) {
      const end = source.indexOf('*/', index + 2)
      if (end < 0) return undefined
      index = end + 2
      continue
    }
    if (char === '"' || char === "'" || char === '`') {
      index++
      while (index < source.length && source[index] !== char) {
        if (char !== '`' && source[index] === '\\') index++
        index++
      }
      if (index >= source.length) return undefined
      index++
    } else if (/[A-Za-z_]/.test(char)) {
      while (index < source.length && /[A-Za-z0-9_]/.test(source[index]!)) index++
    } else if (char.charCodeAt(0) > 127) {
      // 読み取れない識別子を分割して、末尾だけを import 別名と誤認しない。
      return undefined
    } else {
      index++
    }
    tokens.push({ text: source.slice(start, index), start, end: index })
  }
  return tokens
}

const IDENTIFIER = /^[A-Za-z_][A-Za-z0-9_]*$/

function importReferences(source: string) {
  const tokens = tokensOf(source)
  if (!tokens) return undefined
  const aliases = new Map<string, string>()
  const imported = new Set<Token>()
  for (let index = 0; index < tokens.length; index++) {
    if (tokens[index]?.text !== 'import') continue
    index++
    const grouped = tokens[index]?.text === '('
    if (grouped) index++
    do {
      if (tokens[index]?.text === ';') {
        index++
        continue
      }
      if (grouped && tokens[index]?.text === ')') break
      let name: string | undefined
      const alias = tokens[index]
      if (alias && (IDENTIFIER.test(alias.text) || alias.text === '.')) {
        name = alias.text
        imported.add(alias)
        index++
      }
      const literal = tokens[index]?.text
      if (!literal || !['"', '`'].includes(literal[0]!) || literal.includes('\\')) return undefined
      const path = literal.slice(1, -1)
      name ??= path.slice(path.lastIndexOf('/') + 1)
      if (name !== '_' && name !== '.') {
        if (!IDENTIFIER.test(name) || aliases.has(name)) return undefined
        aliases.set(name, path)
      }
      index++
    } while (grouped && index < tokens.length)
    if (!grouped) index--
    else if (tokens[index]?.text !== ')') return undefined
  }

  // 宣言、引数、フィールドなどにも同名があれば、束縛を推測して追従を許可しない。
  for (const [index, token] of tokens.entries()) {
    if (imported.has(token) || !aliases.has(token.text)) continue
    if (tokens[index + 1]?.text !== '.' || tokens[index - 1]?.text === '.') {
      aliases.delete(token.text)
    }
  }
  return { tokens, aliases, imported }
}

/** 両側で同じ名前か参照先を持つ import だけを比較用に正規化する。 */
export function normalizeGoImportAliases(before: string, after: string): [string, string] {
  const original = importReferences(before)
  const current = importReferences(after)
  if (!original || !current) return [before, after]
  const paths = new Set(original.aliases.values())
  const names = new Set(original.aliases.keys())
  const commonPaths = new Set([...current.aliases.values()].filter((path) => paths.has(path)))
  const commonNames = new Set([...current.aliases.keys()].filter((name) => names.has(name)))
  function normalize(source: string, references: NonNullable<ReturnType<typeof importReferences>>) {
    let normalized = ''
    let offset = 0
    for (const token of references.tokens) {
      const path = references.aliases.get(token.text)
      if (!path || references.imported.has(token)) continue
      if (!commonPaths.has(path) && !commonNames.has(token.text)) continue
      normalized += `${source.slice(offset, token.start)}@import(${JSON.stringify(path)})`
      offset = token.end
    }
    return normalized + source.slice(offset)
  }
  return [normalize(before, original), normalize(after, current)]
}
