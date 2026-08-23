/** リポジトリ固有のエージェント向け手順を現在の開発工程に同期させる。 */

export type GuidanceDocument = {
  file: string
  source: string
}

export type AgentGuidanceFinding = {
  file: string
  message: string
}

const requiredMarkers = new Map<string, string[]>([
  [
    '.agents/skills/spec-change/SKILL.md',
    ['spec/contexts/<context>/{models,main}.tsp', 'docs/contexts/<context>/scenarios.md'],
  ],
  [
    '.agents/skills/update-design/SKILL.md',
    ['docs/README.md', 'docs/structure.md', 'docs/contexts/<context>/README.md'],
  ],
  [
    '.agents/skills/implement-work-item/SKILL.md',
    ['risk-based-v2', 'Acceptance RED', 'Unit RED', 'GREEN', 'refactor', 'N/A:', 'actually failed'],
  ],
])

const requiredOrderedMarkers = new Map<string, string[][]>([
  [
    '.agents/skills/implement-work-item/SKILL.md',
    [['Acceptance RED', 'Unit RED', 'GREEN', 'refactor']],
  ],
])

export const agentGuidanceFiles = [...requiredMarkers.keys()]

export function verifyAgentGuidance(documents: GuidanceDocument[]): AgentGuidanceFinding[] {
  const byFile = new Map(documents.map((document) => [document.file, document.source]))
  const findings: AgentGuidanceFinding[] = []

  for (const document of documents) {
    if (document.source.includes('SPECIFICATION.md')) {
      findings.push({
        file: document.file,
        message: 'references the retired SPECIFICATION.md format',
      })
    }
    if (
      document.file === '.agents/skills/implement-work-item/SKILL.md' &&
      document.source.includes('risk-based-v1')
    ) {
      findings.push({ file: document.file, message: 'references the retired risk-based-v1 policy' })
    }
  }

  for (const [file, markers] of requiredMarkers) {
    const source = byFile.get(file) ?? ''
    for (const marker of markers) {
      if (!source.includes(marker)) {
        findings.push({ file, message: `is missing required marker: ${marker}` })
      }
    }
  }

  for (const [file, sequences] of requiredOrderedMarkers) {
    const source = byFile.get(file) ?? ''
    for (const sequence of sequences) {
      let offset = 0
      const inOrder = sequence.every((marker) => {
        const index = source.indexOf(marker, offset)
        if (index < 0) return false
        offset = index + marker.length
        return true
      })
      if (!inOrder && sequence.every((marker) => source.includes(marker))) {
        findings.push({
          file,
          message: `does not preserve required sequence: ${sequence.join(' -> ')}`,
        })
      }
    }
  }

  return findings
}
