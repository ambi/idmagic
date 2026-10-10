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
  ['AGENTS.md', ['コードの編集またはレビュー', 'docs/development/coding-style.md']],
  [
    '.agents/skills/spec-change/SKILL.md',
    [
      'spec/modules/<module>/{models,main}.tsp',
      'docs/modules/<module>/<group>/<feature>/README.md',
      'acceptance.feature.md',
    ],
  ],
  [
    '.agents/skills/update-design/SKILL.md',
    ['docs/README.md', 'docs/design/application/backend.md', 'docs/modules/<module>/README.md'],
  ],
  [
    '.agents/skills/implement-work-item/SKILL.md',
    [
      'risk-based-v4',
      'Acceptance RED',
      'Unit RED',
      'fault_model',
      'GREEN',
      'refactor',
      'N/A:',
      'actually failed',
      'Before the first source or test edit',
      'docs/development/coding-style.md',
      'review the changed code',
    ],
  ],
])

const requiredOrderedMarkers = new Map<string, string[][]>([
  ['.agents/skills/implement-work-item/SKILL.md', [['RED', 'GREEN', 'refactor']]],
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
      /risk-based-v[12]/.test(document.source)
    ) {
      findings.push({ file: document.file, message: 'references a retired evidence policy' })
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
