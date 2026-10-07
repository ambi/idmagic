import { basename } from 'node:path'
import { type Finding, type SCHEMAS, lintRawText, validateAgainstSchema } from './lib.ts'

// 節見出しと記録の項目名の対応 (WORK_ITEM_FORMAT.md)。フォーマット文書は日本語の
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
  ['completion', 'completion'],
  ['完了', 'completion'],
])

// 完了節のフィールド名と記録の項目名の対応 (WORK_ITEM_FORMAT.md)。節見出しと同じく、
// フォーマット文書が示す日本語と既存の記録が使う英語を同じ項目へ解決する。
const COMPLETION_FIELDS = new Map<string, string>([
  ['completed at', 'completed_at'],
  ['完了日', 'completed_at'],
  ['summary', 'summary'],
  ['要約', 'summary'],
  ['verification results', 'verification'],
  ['検証結果', 'verification'],
  ['red evidence', 'red_evidence'],
  ['red の証拠', 'red_evidence'],
  ['acceptance red evidence', 'acceptance_red_evidence'],
  ['受け入れ red の証拠', 'acceptance_red_evidence'],
  ['unit red evidence', 'unit_red_evidence'],
  ['単体 red の証拠', 'unit_red_evidence'],
  ['primary use case evidence', 'primary_use_case_evidence'],
  ['主要ユースケースの証拠', 'primary_use_case_evidence'],
  ['independent verification', 'independent_verification'],
  ['独立した検証', 'independent_verification'],
  ['change-resistance results', 'change_resistance'],
  ['変更耐性の結果', 'change_resistance'],
])

const RED_EVIDENCE_FIELDS = new Map<string, string>([
  ['test', 'test'],
  ['テスト', 'test'],
  ['requirement', 'requirement'],
  ['要件', 'requirement'],
  ['observed failure', 'observed_failure'],
  ['観測した失敗', 'observed_failure'],
  ['detection reason', 'detection_reason'],
  ['検出できる理由', 'detection_reason'],
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

  // id is not authored in frontmatter (WORK_ITEM_FORMAT.md): it is always the
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
      } else if (sectionKey === 'completion') {
        const completion: Record<string, unknown> = {}
        const compLines = content.split('\n')
        let currentField = ''
        let currentText: string[] = []

        const flushField = () => {
          if (currentField) {
            if (currentField === 'verification') {
              completion.verification = currentText
                .map((l) => l.replace(/^-\s*/, '').trim())
                .filter(Boolean)
            } else if (
              currentField === 'red_evidence' ||
              currentField === 'acceptance_red_evidence' ||
              currentField === 'unit_red_evidence'
            ) {
              const redEvidence: Record<string, string> = {}
              for (const line of currentText) {
                const field = line.match(/^-\s+\*\*([^*]+)\*\*:\s*(.*)$/)
                const key = field?.[1]
                  ? RED_EVIDENCE_FIELDS.get(field[1].trim().toLowerCase())
                  : undefined
                if (key && field?.[2]) redEvidence[key] = field[2].trim()
              }
              completion[currentField] = redEvidence
            } else if (currentField === 'primary_use_case_evidence') {
              completion.primary_use_case_evidence = Bun.YAML.parse(currentText.join('\n'))
            } else {
              completion[currentField] = currentText.join('\n').trim()
            }
            currentField = ''
            currentText = []
          }
        }

        for (const line of compLines) {
          const m = line.match(/^-\s+\*\*([^*]+)\*\*:\s*(.*)$/)
          if (m?.[1]) {
            flushField()
            const field = COMPLETION_FIELDS.get(m[1].trim().toLowerCase())
            const value = m[2] ? m[2].trim() : ''
            if (field === 'completed_at') {
              completion.completed_at = value
            } else if (field) {
              currentField = field
              if (value) currentText.push(value)
            }
          } else if (currentField) {
            currentText.push(line.replace(/^\s{2}/, ''))
          }
        }
        flushField()
        data.completion = completion
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
