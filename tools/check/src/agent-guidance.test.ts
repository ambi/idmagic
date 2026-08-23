import { describe, expect, it } from 'bun:test'
import { verifyAgentGuidance } from './agent-guidance.ts'

const currentGuidance = [
  {
    file: '.agents/skills/spec-change/SKILL.md',
    source: 'spec/contexts/<context>/{models,main}.tsp\ndocs/contexts/<context>/scenarios.md',
  },
  {
    file: '.agents/skills/update-design/SKILL.md',
    source: 'docs/README.md\ndocs/structure.md\ndocs/contexts/<context>/README.md',
  },
  {
    file: '.agents/skills/implement-work-item/SKILL.md',
    source: 'risk-based-v2\nAcceptance RED\nUnit RED\nGREEN\nrefactor\nN/A:\nactually failed',
  },
]

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
      message: 'is missing required marker: docs/structure.md',
    })
  })

  it('rejects the retired evidence policy in implementation guidance', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[2] = {
      file: currentGuidance[2]!.file,
      source: `${currentGuidance[2]!.source}\nrisk-based-v1`,
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/implement-work-item/SKILL.md',
      message: 'references the retired risk-based-v1 policy',
    })
  })

  it('rejects implementation guidance that reverses the development loop', () => {
    const guidance = currentGuidance.map((document) => ({ ...document }))
    guidance[2] = {
      file: currentGuidance[2]!.file,
      source: 'risk-based-v2\nUnit RED\nAcceptance RED\nrefactor\nGREEN\nN/A:\nactually failed',
    }
    expect(verifyAgentGuidance(guidance)).toContainEqual({
      file: '.agents/skills/implement-work-item/SKILL.md',
      message:
        'does not preserve required sequence: Acceptance RED -> Unit RED -> GREEN -> refactor',
    })
  })
})
