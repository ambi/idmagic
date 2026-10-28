import { describe, expect, it } from 'bun:test'
import {
  type MaturityEnvironment,
  diffFeatureMaturities,
  verifyMaturityPromotions,
} from './maturity-promotion.ts'

const promotion = { feature: 'demo-v1', from: 'preview', to: 'supported' } as const
const environment = (overrides: Partial<MaturityEnvironment> = {}): MaturityEnvironment => ({
  releasePhase: 'unpublished',
  read: () => undefined,
  maturityChanges: [promotion],
  ...overrides,
})
const completed = {
  id: 'wi-999-promote-demo',
  status: 'completed',
  primary_use_cases: [{}],
  maturity_evidence: [
    {
      ...promotion,
      security: 'セキュリティ確認を実施した。',
      compatibility: '既存データを保持する。',
    },
  ],
}

describe('verifyMaturityPromotions', () => {
  it('derives feature maturity changes from registry definitions', () => {
    const base = `return FeatureRegistry{
      {ID: "demo-v1", Maturity: FeatureExperimental},
      {ID: "removed-v1", Maturity: FeatureSupported},
    }`
    const head = `return FeatureRegistry{
      {ID: "demo-v1", Maturity: FeaturePreview},
      {ID: "new-v1", Maturity: FeatureExperimental},
    }`
    expect(diffFeatureMaturities(base, head)).toEqual([
      { feature: 'demo-v1', from: 'experimental', to: 'preview' },
      { feature: 'new-v1', from: undefined, to: 'experimental' },
      { feature: 'removed-v1', from: 'supported', to: undefined },
    ])
  })

  it('公開前は告知パスだけを省略でき、ほかの証拠は求める', () => {
    expect(verifyMaturityPromotions(completed, environment())).toEqual([])
    expect(
      verifyMaturityPromotions(
        { ...completed, primary_use_cases: [], maturity_evidence: [] },
        environment(),
      ),
    ).toEqual([
      'primary_use_cases is required for maturity promotion demo-v1: preview -> supported',
      'maturity_evidence is required for promotion demo-v1: preview -> supported',
    ])
    expect(
      verifyMaturityPromotions(
        { ...completed, maturity_evidence: [{ ...promotion }] },
        environment(),
      ),
    ).toEqual([
      'maturity_evidence demo-v1 has no security result',
      'maturity_evidence demo-v1 has neither compatibility nor migration information',
    ])
  })

  it('公開後は昇格を示す告知文書を求める', () => {
    const published = environment({ releasePhase: 'published' })
    expect(verifyMaturityPromotions(completed, published)).toEqual([
      'maturity_evidence demo-v1 has no documentation path',
    ])
    const documentation = 'docs/releases/changes/wi-999-promote-demo.md'
    const withDocument = {
      ...completed,
      maturity_evidence: [{ ...completed.maturity_evidence[0], documentation }],
    }
    expect(
      verifyMaturityPromotions(withDocument, environment({ read: () => '# Release\n' })),
    ).toEqual([`maturity documentation ${documentation} does not show demo-v1 as supported`])
  })

  it('この作業ツリーが変えていない完了記録には昇格を帰属させない', () => {
    expect(
      verifyMaturityPromotions(
        { ...completed, maturity_evidence: [] },
        environment({ changedRecords: new Set(['wi-1-other']) }),
      ),
    ).toEqual([])
  })
})
