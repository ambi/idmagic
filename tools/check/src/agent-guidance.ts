/** リポジトリ固有のエージェント向け手順を現在の開発工程に同期させる。 */

import { directTasks } from './verification-tasks.ts'

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
    'AGENTS.md',
    ['docs/development/coding-style.md', 'docs/development/task-context.md', '--target'],
  ],
  [
    '.agents/skills/spec-change/SKILL.md',
    [
      'spec/modules/<module>/{models,main}.tsp',
      'docs/modules/<module>/<group>/<feature>/README.md',
      'acceptance.feature.md',
      'docs/development/task-context.md',
      '--concern contract',
    ],
  ],
  [
    '.agents/skills/update-design/SKILL.md',
    [
      'docs/README.md',
      'docs/design/application/backend.md',
      'docs/modules/<module>/README.md',
      'docs/development/task-context.md',
      '--concern boundary',
    ],
  ],
  [
    '.agents/skills/implement-work-item/SKILL.md',
    [
      'fault_model',
      'docs/development/coding-style.md',
      'docs/development/task-context.md',
      '--target',
    ],
  ],
])

export const agentGuidanceFiles = [...requiredMarkers.keys()]

export function verifyAgentGuidance(
  documents: GuidanceDocument[],
  declaredTasks: ReadonlySet<string>,
): AgentGuidanceFinding[] {
  const byFile = new Map(documents.map((document) => [document.file, document.source]))
  const findings: AgentGuidanceFinding[] = []

  for (const document of documents) {
    // 同じコマンドの読み方を用い、-c、改行継続、::: の後のタスクも照合する。
    const invoked = directTasks({ guidance: { run: document.source } }, 'guidance')
    for (const task of invoked) {
      if (!declaredTasks.has(task)) {
        findings.push({ file: document.file, message: `calls an undeclared mise task: ${task}` })
      }
    }
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
    if (!directTasks({ guidance: { run: source } }, 'guidance').includes('brief')) {
      findings.push({ file, message: 'is missing required mise task: brief' })
    }
    for (const marker of markers) {
      if (!source.includes(marker)) {
        findings.push({ file, message: `is missing required marker: ${marker}` })
      }
    }
  }

  return findings
}
