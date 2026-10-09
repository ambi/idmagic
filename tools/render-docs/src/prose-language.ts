/**
 * 生成サイトの本文に残った英文を見つける。
 *
 * 文章の言語（docs/development/writing-language.md）は、人が読む文章を日本語とし、識別子、
 * 規格名、製品名などは原表記を保つと定める。ASCII の有無では両者を区別できないので、
 * ラテン文字の単語が 3 語以上続き、その中に英語の機能語を含む並びを英文とみなす。
 * 規格名や書名のように機能語を含む原表記は、許容する理由とともに ALLOWED_ENGLISH に載せる。
 */

export type EnglishProse = { page: string; text: string }

/**
 * 許容する理由の分類。規格名、文書名、製品名は外部が定める名前、記法は仕様の形式が定めて
 * 検査器が読む字句、用語は日本語の定訳がなく原表記で通用する術語や別名である。
 */
export type AllowedEnglish = {
  phrase: string
  reason: '規格名' | '文書名' | '製品名' | '記法' | '用語'
}

export const ALLOWED_ENGLISH: readonly AllowedEnglish[] = [
  { phrase: 'Proof Key for Code Exchange by OAuth Public Clients', reason: '規格名' },
  { phrase: 'Proof Key for Code Exchange', reason: '規格名' },
  { phrase: 'The OAuth 2.0 Authorization Framework Bearer Token Usage', reason: '規格名' },
  { phrase: 'The OAuth 2.0 Authorization Framework', reason: '規格名' },
  { phrase: 'OAuth 2.0 Demonstrating Proof of Possession', reason: '規格名' },
  { phrase: 'Demonstrating Proof of Possession', reason: '規格名' },
  { phrase: 'Proof of Possession', reason: '規格名' },
  { phrase: 'Proof-of-Possession Key Semantics for JSON Web Tokens', reason: '規格名' },
  { phrase: 'JSON Web Token Profile for OAuth 2.0 Access Tokens', reason: '規格名' },
  {
    phrase: 'JSON Web Token Profile for OAuth 2.0 Client Authentication and Authorization Grants',
    reason: '規格名',
  },
  {
    phrase: 'OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens',
    reason: '規格名',
  },
  { phrase: 'Best Current Practice for OAuth 2.0 Security', reason: '規格名' },
  { phrase: 'Resource Indicators for OAuth 2.0', reason: '規格名' },
  { phrase: 'OAuth 2.0 and OpenID Connect', reason: '規格名' },
  { phrase: 'Subject Identifiers for Security Event Tokens', reason: '規格名' },
  { phrase: 'System for Cross-domain Identity Management Protocol', reason: '規格名' },
  { phrase: 'System for Cross-domain Identity Management Core Schema', reason: '規格名' },
  {
    phrase: 'Assertions and Protocols for the OASIS Security Assertion Markup Language',
    reason: '規格名',
  },
  { phrase: 'Bindings for the OASIS Security Assertion Markup Language', reason: '規格名' },
  { phrase: 'Profiles for the OASIS Security Assertion Markup Language', reason: '規格名' },
  { phrase: 'Metadata for the OASIS Security Assertion Markup Language', reason: '規格名' },
  { phrase: 'Enhanced Client or Proxy Profile', reason: '規格名' },
  { phrase: 'Authentication and Authenticator Management', reason: '規格名' },
  { phrase: 'An API for accessing Public Key Credentials Level 3', reason: '規格名' },
  { phrase: 'Access Request and Approval Profile', reason: '規格名' },
  { phrase: 'ECDSA using P-256 and SHA-256', reason: '規格名' },
  { phrase: 'Easy Approach to Requirements Syntax', reason: '規格名' },
  { phrase: 'Markdown with Gherkin', reason: '規格名' },
  // STRIDE の分類名。脅威モデルの表で分類の値として使う。
  { phrase: 'Elevation of privilege', reason: '規格名' },
  { phrase: 'Denial of service', reason: '規格名' },
  { phrase: 'The Art of Readable Code', reason: '文書名' },
  { phrase: 'Working Effectively with Legacy Code', reason: '文書名' },
  { phrase: 'A Philosophy of Software Design', reason: '文書名' },
  { phrase: 'Threat Modeling: Designing for Security', reason: '文書名' },
  { phrase: 'Software Engineering at Google', reason: '文書名' },
  { phrase: 'The Clean Architecture', reason: '文書名' },
  {
    phrase: 'QuickCheck: A Lightweight Tool for Random Testing of Haskell Programs',
    reason: '文書名',
  },
  { phrase: 'A Rational Design Process: How and Why to Fake It', reason: '文書名' },
  { phrase: 'INCOSE Guide to Writing Requirements', reason: '文書名' },
  { phrase: 'Tests and Requirements, Requirements and Tests: A Möbius Strip', reason: '文書名' },
  { phrase: 'Modular monolith and package by component', reason: '文書名' },
  { phrase: 'PSScriptAnalyzer rules and recommendations', reason: '文書名' },
  { phrase: 'Cloud SQL for PostgreSQL', reason: '製品名' },
  { phrase: 'Managed Service for Prometheus', reason: '製品名' },
  // 廃止した規則の見出しに付ける `(superseded by REQ-…)`（docs/formats/specification-format.md）。
  { phrase: 'superseded by', reason: '記法' },
  { phrase: 'Ports and Adapters', reason: '用語' },
  // 他製品が同じ機能に付ける名前。用語集の別名の列に載る。
  { phrase: 'remember this device', reason: '用語' },
]

/** 短い語句が長い語句の一部を先に切り取らないように、長い順に適用する。 */
const ALLOWED_PHRASES = ALLOWED_ENGLISH.map(({ phrase }) => phrase).sort(
  (left, right) => right.length - left.length,
)

/** 中身を本文として読まない要素。図やコードの中の英語は文章ではない。 */
const OPAQUE_BLOCKS = /<(pre|script|style|svg|textarea)\b[^>]*>[\s\S]*?<\/\1>/gi

/** 文の途中に現れる要素。境界にすると、リンクや強調を挟んだ英文を見逃す。 */
const INLINE_TAGS =
  /<\/?(a|em|strong|b|i|span|abbr|sub|sup|small|mark|kbd|q|cite|dfn|var)\b[^>]*>/gi

/** コードは識別子なので読まないが、文の途中にあるため境界にはしない。 */
const INLINE_CODE = /<code\b[^>]*>[\s\S]*?<\/code>/gi

const BOUNDARY = '\u0000'

const WORD_RUN = /[A-Za-z][\w'’./-]*(?:(?:\s+|[,;:]\s*)[A-Za-z0-9][\w'’./-]*)*/g

const FUNCTION_WORDS = new Set([
  'a',
  'an',
  'and',
  'are',
  'as',
  'at',
  'be',
  'been',
  'but',
  'by',
  'can',
  'does',
  'each',
  'every',
  'for',
  'from',
  'has',
  'have',
  'if',
  'in',
  'into',
  'is',
  'it',
  'its',
  'must',
  'not',
  'of',
  'on',
  'only',
  'or',
  'should',
  'than',
  'that',
  'the',
  'this',
  'to',
  'was',
  'were',
  'when',
  'which',
  'will',
  'with',
  'A',
  'An',
  'Each',
  'Every',
  'If',
  'It',
  'Only',
  'The',
  'This',
])

const ENTITIES: Record<string, string> = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'" }

function decodeEntities(text: string): string {
  return text.replace(/&(#x[0-9a-f]+|#\d+|[a-z]+);/gi, (entity, name: string) => {
    if (name.startsWith('#x') || name.startsWith('#X'))
      return String.fromCodePoint(Number.parseInt(name.slice(2), 16))
    if (name.startsWith('#')) return String.fromCodePoint(Number.parseInt(name.slice(1), 10))
    return ENTITIES[name.toLowerCase()] ?? entity
  })
}

function textSegments(html: string): string[] {
  const text = html
    .replace(OPAQUE_BLOCKS, BOUNDARY)
    .replace(INLINE_CODE, ' ')
    .replace(INLINE_TAGS, '')
    .replace(/<[^>]*>/g, BOUNDARY)
  let segments = decodeEntities(text).split(BOUNDARY)
  for (const phrase of ALLOWED_PHRASES)
    segments = segments.flatMap((segment) => segment.split(phrase))
  return segments
}

function isEnglishSentence(run: string): boolean {
  const words = run.split(/[\s,;:]+/).filter(Boolean)
  return words.length >= 3 && words.some((word) => FUNCTION_WORDS.has(word.replace(/[.]+$/, '')))
}

export function findEnglishProse(pages: Record<string, string>): EnglishProse[] {
  const findings: EnglishProse[] = []
  for (const [page, html] of Object.entries(pages)) {
    if (!page.endsWith('.html')) continue
    for (const segment of textSegments(html)) {
      for (const [run] of segment.matchAll(WORD_RUN)) {
        const text = run.replace(/\s+/g, ' ').trim()
        if (isEnglishSentence(text)) findings.push({ page, text })
      }
    }
  }
  return findings
}
