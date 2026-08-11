/** Parse ARCHITECTURE.md frontmatter and known prose sections for schema checks. */

const BODY_HEADINGS: Record<string, string> = {
  overview: 'overview',
  structure: 'structure',
  stack: 'stack',
  'context map': 'context_map',
  conventions: 'conventions',
  'cross-cutting concerns': 'cross_cutting_concerns',
  'runtime composition': 'runtime_composition',
  'structural decisions': 'structural_decisions',
  'documentation policy': 'documentation_policy',
  'design decisions': 'design_decisions',
  diagrams: 'diagrams',
}

export function parseArchitectureDoc(text: string): Record<string, unknown> {
  const match = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n([\s\S]*)$/)
  let frontmatter: Record<string, unknown> = {}
  let body = text
  if (match?.[1] !== undefined && match[2] !== undefined) {
    const parsed = Bun.YAML.parse(match[1])
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      frontmatter = parsed as Record<string, unknown>
    }
    body = match[2]
  }

  const sections: Record<string, unknown> = {}
  for (const section of body.split(/(?=^#{1,2}\s+)/m)) {
    const lines = section.split('\n')
    const heading = (lines[0] ?? '')
      .match(/^#{1,2}\s+(.+)$/)?.[1]
      ?.trim()
      .toLowerCase()
    if (!heading) continue
    const key = BODY_HEADINGS[heading]
    if (!key) continue
    const content = lines.slice(1).join('\n').trim()
    if (content) sections[key] = content
  }
  return { ...sections, ...frontmatter }
}
