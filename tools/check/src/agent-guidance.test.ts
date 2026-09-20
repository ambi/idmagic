import { describe, expect, it } from 'bun:test'
import { verifyAgentGuidance } from './agent-guidance.ts'

const currentGuidance = [
  {
    file: 'AGENTS.md',
    source: 'コードの編集またはレビューでは docs/development/coding-style.md を読む。',
  },
  {
    file: '.agents/skills/spec-change/SKILL.md',
    source: 'spec/contexts/<context>/{models,main}.tsp\ndocs/domain/<context>/scenarios.feature.md',
  },
  {
    file: '.agents/skills/update-design/SKILL.md',
    source: 'docs/README.md\ndocs/domain/structure.md\ndocs/domain/<context>/README.md',
  },
  {
    file: '.agents/skills/implement-work-item/SKILL.md',
    source:
      'Before the first source or test edit, read docs/development/coding-style.md.\nrisk-based-v3\nAcceptance RED\nUnit RED\nE2E RED\nGREEN\nrefactor\nN/A:\nactually failed\nreview the changed code',
  },
]

const rootGuidanceIndex = currentGuidance.findIndex((document) => document.file === 'AGENTS.md')
const implementationGuidanceIndex = currentGuidance.findIndex(
  (document) => document.file === '.agents/skills/implement-work-item/SKILL.md',
)

describe('verifyAgentGuidance', () => {
  it('accepts guidance that names the current sources and development loop', () => {
    expect(verifyAgentGuidance(currentGuidance)).toEqual([])
  })

  it('rejects a legacy monolithic specification pointer', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[0] = {
      file: '.agents/skills/spec-change/SKILL.md',
      source: 'Update spec/contexts/<context>/SPECIFICATION.md.',
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/spec-change/SKILL.md',
      message: 'references the retired SPECIFICATION.md format',
    })
  })

  it('rejects guidance missing a required current marker', () => {
    const guidance = currentGuidance.filter(
      (document) => document.file !== '.agents/skills/update-design/SKILL.md',
    )
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/update-design/SKILL.md',
      message: 'is missing required marker: docs/domain/structure.md',
    })
  })

  it('rejects root guidance that does not route code work to the coding style', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[rootGuidanceIndex] = {
      file: 'AGENTS.md',
      source: 'docs/development/coding-style.md を読む。',
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: 'AGENTS.md',
      message: 'is missing required marker: コードの編集またはレビュー',
    })
  })

  it('rejects implementation guidance that does not load the coding style', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[implementationGuidanceIndex] = {
      file: currentGuidance[implementationGuidanceIndex]!.file,
      source: currentGuidance[implementationGuidanceIndex]!.source.replace(
        'Before the first source or test edit, read ',
        '',
      ),
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/implement-work-item/SKILL.md',
      message: 'is missing required marker: Before the first source or test edit',
    })
  })

  it('rejects the retired evidence policy in implementation guidance', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[implementationGuidanceIndex] = {
      file: currentGuidance[implementationGuidanceIndex]!.file,
      source: `${currentGuidance[implementationGuidanceIndex]!.source}\nrisk-based-v1`,
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/implement-work-item/SKILL.md',
      message: 'references a retired evidence policy',
    })
  })

  it('rejects implementation guidance that reverses the development loop', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[implementationGuidanceIndex] = {
      file: currentGuidance[implementationGuidanceIndex]!.file,
      source:
        'risk-based-v3\nUnit RED\nAcceptance RED\nE2E RED\nrefactor\nGREEN\nN/A:\nactually failed',
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/implement-work-item/SKILL.md',
      message:
        'does not preserve required sequence: Acceptance RED -> Unit RED -> GREEN -> refactor',
    })
  })
})
