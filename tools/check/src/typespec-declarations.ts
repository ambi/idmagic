/**
 * TypeSpec のソースから、宣言ごとの名前と本文を取り出す。
 *
 * `spec-diff` は履歴のリビジョンを文字列のまま比べるので、コンパイラーを通さずに読む。
 * 本文は、直前のデコレーターから宣言の終わりまでを、コメントを除いて字句ごとに空白一つで
 * つないだものである。整形だけの変更は本文を変えず、項目、型、デコレーターの変更は変える。
 */

export type TypeSpecDeclaration = { name: string; text: string }

type Token = { text: string; kind: 'word' | 'string' | 'punctuation' }

const DECLARATION_KEYWORDS = new Set(['alias', 'enum', 'model', 'op', 'scalar', 'union'])
const CONTAINER_KEYWORDS = new Set(['namespace', 'interface'])
/** 本体のブロックで終わり、セミコロンを伴わない宣言。 */
const BLOCK_BODIED_KEYWORDS = new Set(['enum', 'model', 'union'])
/** 閉じ括弧の後にこれらが続く間は、型の式がまだ終わっていない。 */
const CONTINUES_AFTER_BLOCK = new Set(['>', '|', '&', ',', ';', ')'])
const OPENERS = new Set(['(', '[', '{'])
const CLOSERS = new Set([')', ']', '}'])
const WORD = /[A-Za-z_$][A-Za-z0-9_$]*/y

function tokenize(source: string): Token[] {
  const tokens: Token[] = []
  let index = 0
  while (index < source.length) {
    const char = source[index]!
    if (/\s/.test(char)) {
      index += 1
    } else if (source.startsWith('//', index)) {
      const end = source.indexOf('\n', index)
      index = end === -1 ? source.length : end
    } else if (source.startsWith('/*', index)) {
      const end = source.indexOf('*/', index + 2)
      index = end === -1 ? source.length : end + 2
    } else if (source.startsWith('"""', index)) {
      const end = source.indexOf('"""', index + 3)
      const stop = end === -1 ? source.length : end + 3
      tokens.push({ text: source.slice(index, stop), kind: 'string' })
      index = stop
    } else if (char === '"') {
      let stop = index + 1
      while (stop < source.length && source[stop] !== '"') stop += source[stop] === '\\' ? 2 : 1
      stop = Math.min(stop + 1, source.length)
      tokens.push({ text: source.slice(index, stop), kind: 'string' })
      index = stop
    } else {
      WORD.lastIndex = index
      const word = WORD.exec(source)?.[0]
      tokens.push(word ? { text: word, kind: 'word' } : { text: char, kind: 'punctuation' })
      index += word?.length ?? 1
    }
  }
  return tokens
}

function isPunctuation(token: Token | undefined, text: string): boolean {
  return token?.kind === 'punctuation' && token.text === text
}

/** `index` の開き括弧に対応する閉じ括弧の次の位置を返す。 */
function afterBalanced(tokens: readonly Token[], index: number): number {
  let depth = 0
  for (let position = index; position < tokens.length; position += 1) {
    const token = tokens[position]!
    if (token.kind !== 'punctuation') continue
    if (OPENERS.has(token.text)) depth += 1
    else if (CLOSERS.has(token.text)) {
      depth -= 1
      if (depth === 0) return position + 1
    }
  }
  return tokens.length
}

/** デコレーターとディレクティブを読み飛ばし、文のキーワードの位置を返す。 */
function afterDecorators(tokens: readonly Token[], index: number): number {
  let position = index
  for (;;) {
    if (isPunctuation(tokens[position], '@')) {
      position += 1
      while (isPunctuation(tokens[position], '@')) position += 1
      if (tokens[position]?.kind === 'word') position += 1
      while (isPunctuation(tokens[position], '.') && tokens[position + 1]?.kind === 'word') {
        position += 2
      }
      if (isPunctuation(tokens[position], '(')) position = afterBalanced(tokens, position)
    } else if (isPunctuation(tokens[position], '#')) {
      position += 1
      if (tokens[position]?.kind === 'word') position += 1
      while (tokens[position]?.kind === 'string') position += 1
    } else {
      return position
    }
  }
}

/** 文の最後の字句の位置を返す。 */
function statementEnd(tokens: readonly Token[], keywordIndex: number): number {
  const blockBodied = BLOCK_BODIED_KEYWORDS.has(tokens[keywordIndex]?.text ?? '')
  let depth = 0
  for (let position = keywordIndex; position < tokens.length; position += 1) {
    const token = tokens[position]!
    if (token.kind !== 'punctuation') continue
    if (OPENERS.has(token.text)) depth += 1
    else if (CLOSERS.has(token.text)) {
      // 対応する開きのない閉じ括弧は、この文を含むコンテナーのものである。
      if (depth === 0) return position - 1
      depth -= 1
      const next = tokens[position + 1]
      const continues = next?.kind === 'punctuation' && CONTINUES_AFTER_BLOCK.has(next.text)
      if (depth === 0 && token.text === '}' && blockBodied && !continues) return position
    } else if (token.text === ';' && depth === 0) {
      return position
    }
  }
  return tokens.length - 1
}

export function typeSpecDeclarations(source: string): TypeSpecDeclaration[] {
  const tokens = tokenize(source)
  const declarations: TypeSpecDeclaration[] = []
  let index = 0
  while (index < tokens.length) {
    if (isPunctuation(tokens[index], '}')) {
      index += 1
      continue
    }
    const keywordIndex = afterDecorators(tokens, index)
    const keyword = tokens[keywordIndex]
    if (keyword?.kind === 'word' && CONTAINER_KEYWORDS.has(keyword.text)) {
      // コンテナーは本体を開くだけで、内側の文は同じ走査で読む。
      let position = keywordIndex
      while (
        position < tokens.length &&
        !isPunctuation(tokens[position], '{') &&
        !isPunctuation(tokens[position], ';')
      ) {
        position += 1
      }
      index = position + 1
      continue
    }
    const end = Math.max(statementEnd(tokens, keywordIndex), keywordIndex)
    const name = tokens[keywordIndex + 1]
    if (
      keyword?.kind === 'word' &&
      DECLARATION_KEYWORDS.has(keyword.text) &&
      name?.kind === 'word'
    ) {
      declarations.push({
        name: name.text,
        text: tokens
          .slice(index, end + 1)
          .map((token) => token.text)
          .join(' '),
      })
    }
    index = end + 1
  }
  return declarations
}
