import { basename } from 'node:path'
import { type Finding, type SCHEMAS, lintRawText, validateAgainstSchema } from './lib.ts'

// 節見出しと記録の項目名の対応 (docs/formats/work-item-format.md)。フォーマット文書は日本語の
// 見出しを示し、既存の記録は英語の見出しで書かれているため、どちらの表記も同じ項目
// へ解決する。対応表にない見出し (`## Design` / `## 設計` など) は記録の項目を作らず、
// 本文としてそのまま残る。
const SECTION_KEYS = new Map<string, string>([
  ['motivation', 'motivation'],
  ['動機', 'motivation'],
  ['scope', 'scope'],
  ['対象範囲', 'scope'],
  ['out of scope', 'out_of_scope'],
  ['対象外', 'out_of_scope'],
  ['plan', 'plan'],
  ['計画', 'plan'],
  ['tasks', 'tasks'],
  ['タスク', 'tasks'],
  ['verification', 'verification'],
  ['検証', 'verification'],
  ['risk notes', 'risk_notes'],
  ['リスク', 'risk_notes'],
])

export function parseFrontmatterAndMarkdown(path: string, text: string): Record<string, unknown> {
  const match = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n([\s\S]*)$/)
  const data: Record<string, unknown> = {}
  let bodyText = text

  if (match && match[1] !== undefined && match[2] !== undefined) {
    const yamlText = match[1]
    bodyText = match[2]

    const frontmatter = Bun.YAML.parse(yamlText)
    if (frontmatter !== null && typeof frontmatter === 'object' && !Array.isArray(frontmatter)) {
      Object.assign(data, frontmatter)
    }
  }

  // id is not authored in frontmatter (docs/formats/work-item-format.md): it is always the
  // filename stem.
  if (typeof data.id !== 'string' || data.id.length === 0) {
    data.id = basename(path).replace(/\.md$/, '')
  }

  // title is not authored in frontmatter either: it is the body's first
  // heading, unless that heading is actually a known section name (an older,
  // title-less document that starts straight with `# Motivation`).
  if (typeof data.title !== 'string' || data.title.length === 0) {
    const firstHeading = bodyText.match(/^#{1,2}\s+(.+)$/m)
    const headingText = firstHeading?.[1]?.trim()
    if (firstHeading && headingText && !SECTION_KEYS.has(headingText.toLowerCase())) {
      data.title = headingText
      bodyText = bodyText.replace(firstHeading[0], '')
    }
  }

  // Parse markdown headers. Records use a single H1 title followed by H2
  // section headings; older records used H1 for every section. Accept both
  // so the H2 migration is backward compatible.
  const sections = bodyText.split(/(?=^#{1,2}\s+)/m)
  for (const sec of sections) {
    const lines = sec.split('\n')
    const headerLine = lines[0] ?? ''
    const headerMatch = headerLine.match(/^#{1,2}\s+(.+)$/)
    if (headerMatch?.[1]) {
      const sectionKey = SECTION_KEYS.get(headerMatch[1].trim().toLowerCase())
      const content = lines.slice(1).join('\n').trim()
      if (!sectionKey || !content) continue

      if (sectionKey === 'motivation') {
        data.motivation = content
      } else if (sectionKey === 'scope') {
        data.scope = content
      } else if (sectionKey === 'out_of_scope') {
        data.out_of_scope = content
          .split('\n')
          .map((l) => l.replace(/^-\s*/, '').trim())
          .filter(Boolean)
      } else if (sectionKey === 'plan') {
        data.plan = content
      } else if (sectionKey === 'tasks') {
        data.tasks = content
      } else if (sectionKey === 'verification') {
        data.verification = content
          .split('\n')
          .map((l) => l.replace(/^-\s*/, '').trim())
          .filter(Boolean)
      } else if (sectionKey === 'risk_notes') {
        data.risk_notes = content
      }
    }
  }

  return data
}

export type RecordValidation = {
  data?: Record<string, unknown>
  findings: Finding[]
}

/** Markdown 記録を一度だけ解析し、書式と指定 schema の所見を同じ結果へまとめる。 */
export function validateMarkdownRecord(
  path: string,
  text: string,
  schema: keyof typeof SCHEMAS,
): RecordValidation {
  const findings = [...lintRawText(text)]
  let data: Record<string, unknown>
  try {
    data = parseFrontmatterAndMarkdown(path, text)
  } catch (error) {
    findings.unshift({
      line: 1,
      column: 1,
      message: `Failed to parse markdown frontmatter: ${String(error)}`,
    })
    return { findings }
  }
  findings.push(...validateAgainstSchema(schema, data, text))
  return { data, findings }
}
