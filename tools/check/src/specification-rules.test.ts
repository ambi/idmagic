import { describe, expect, it } from 'bun:test'
import {
  featureSlices,
  goDeclarations,
  verifyFeatureNodes,
  verifyRuleFields,
  verifySectionOrder,
} from './specification-rules.ts'

const scenario = (...body: string[]): string =>
  [
    '# Feature: 信頼済みデバイス',
    '',
    '## 有効性',
    '',
    '### Rule: REQ-DEMO-001 両方の期限を満たす間だけ有効とする',
    '',
    ...body,
    '',
    '#### Example: EX-DEMO-001-01 期限ちょうどで無効',
    '',
    '- When 期限ちょうどに照合する',
    '- Then 無効とする',
    '',
  ].join('\n')

const declarations = goDeclarations([
  {
    path: 'backend/demo/trusteddevice/domain/trusted_device.go',
    source: [
      'package domain',
      '',
      'type TrustedDevice struct{}',
      '',
      'func (d *TrustedDevice) Active(now time.Time) bool { return true }',
      '',
      'func NewTrustedDevice() TrustedDevice { return TrustedDevice{} }',
      '',
      'const (',
      '\tMaxAge = 90',
      ')',
    ].join('\n'),
  },
])

const noLinks = (_from: string, _target: string): boolean => false
const messages = (source: string, resolve = noLinks) =>
  verifyRuleFields(
    'docs/domain/demo/trusted-device/scenarios.feature.md',
    source,
    declarations,
    resolve,
  ).map((finding) => finding.message)

describe('verifyRuleFields', () => {
  it('accepts guarantees that name a method, a package member, or a bare declaration', () => {
    expect(
      messages(
        scenario(
          '- 絶対期限は発行時刻に最大有効期間を加えた時刻とする。',
          '- **担保手段**：`TrustedDevice.Active`、`domain.NewTrustedDevice`、`MaxAge`',
        ),
      ),
    ).toEqual([])
  })

  it('rejects a guarantee that names no declaration in the code', () => {
    expect(messages(scenario('- 規則文。', '- 担保手段：`TrustedDevice.Expired`'))).toEqual([
      'REQ-DEMO-001 guarantee `TrustedDevice.Expired` is not declared in backend/',
    ])
  })

  it('reads a field label written in bold like the design documents', () => {
    expect(messages(scenario('- 規則文。', '- **担保手段**：`TrustedDevice.Expired`'))).toEqual([
      'REQ-DEMO-001 guarantee `TrustedDevice.Expired` is not declared in backend/',
    ])
  })

  it('rejects a guarantee that names a method on the wrong type', () => {
    expect(messages(scenario('- 規則文。', '- 担保手段：`LoginSession.Active`'))).toEqual([
      'REQ-DEMO-001 guarantee `LoginSession.Active` is not declared in backend/',
    ])
  })

  it('requires a parent requirement to be a link that resolves', () => {
    const resolveOnly = (target: string) => (from: string, link: string) =>
      from.endsWith('scenarios.feature.md') && link === target
    const parent =
      '- 上位の要件：[ページサイズ](../../../design/application/api-guidelines.md#ページサイズ)'
    expect(
      messages(
        scenario(parent, '- 既定値は 10 件とする。'),
        resolveOnly('../../../design/application/api-guidelines.md#ページサイズ'),
      ),
    ).toEqual([])
    expect(messages(scenario(parent, '- 既定値は 10 件とする。'))).toEqual([
      'REQ-DEMO-001 parent requirement does not resolve: ../../../design/application/api-guidelines.md#ページサイズ',
    ])
    expect(messages(scenario('- 上位の要件：ページサイズ', '- 既定値は 10 件とする。'))).toEqual([
      'REQ-DEMO-001 parent requirement must be a Markdown link to the requirement it departs from',
    ])
  })

  it('rejects vague wording in a statement, a table row, and an open question', () => {
    expect(
      messages(
        scenario(
          '- 期限を適切に判定する。',
          '| 条件 | 結果 |',
          '| --- | --- |',
          '| 期限切れなど | 無効 |',
          '- **要判断**：必要に応じて見直す。',
        ),
      ),
    ).toEqual([
      'REQ-DEMO-001 uses vague wording 「適切に」; state the condition it stands for',
      'REQ-DEMO-001 uses vague wording 「など」; state the condition it stands for',
      'REQ-DEMO-001 uses vague wording 「必要に応じて」; state the condition it stands for',
    ])
  })

  it('leaves vague wording inside code and in the reason field alone', () => {
    expect(
      messages(
        scenario('- `など` という識別子を拒否する。', '- 理由：必要に応じて変える判断である。'),
      ),
    ).toEqual([])
  })
})

describe('verifySectionOrder', () => {
  const featureNode = 'docs/domain/demo/trusted-device/scenarios.feature.md'
  const rule = (id: string) => [
    `### Rule: ${id} 規則`,
    '',
    `#### Example: ${id.replace('REQ', 'EX')}-01 例`,
    '',
    '- When 操作する',
    '- Then 結果になる',
    '',
  ]
  const document = (...sections: Array<[string, string]>) =>
    [
      '# Feature: 機能',
      '',
      ...sections.flatMap(([heading, id]) => [`## ${heading}`, '', ...rule(id)]),
    ].join('\n')

  it('accepts lifecycle sections in lifecycle order', () => {
    expect(
      verifySectionOrder(
        featureNode,
        document(['生成', 'REQ-DEMO-001'], ['失効と変更', 'REQ-DEMO-002']),
      ),
    ).toEqual([])
  })

  it('accepts API sections in API order', () => {
    expect(
      verifySectionOrder(featureNode, document(['入力', 'REQ-DEMO-001'], ['拒否', 'REQ-DEMO-002'])),
    ).toEqual([])
  })

  it('rejects sections out of order', () => {
    expect(
      verifySectionOrder(
        featureNode,
        document(['失効と変更', 'REQ-DEMO-001'], ['生成', 'REQ-DEMO-002']),
      ).map((finding) => finding.message),
    ).toEqual(['section 生成 must come before 失効と変更'])
  })

  it('rejects a section outside the two vocabularies and a mixture of both', () => {
    expect(
      verifySectionOrder(
        featureNode,
        document(['生成', 'REQ-DEMO-001'], ['拒否', 'REQ-DEMO-002']),
      ).map((finding) => finding.message),
    ).toEqual(['section 拒否 mixes the API sections into the lifecycle sections'])
    expect(
      verifySectionOrder(featureNode, document(['その他', 'REQ-DEMO-001'])).map(
        (finding) => finding.message,
      ),
    ).toEqual([
      'section その他 is neither a lifecycle section (生成, 有効性, 利用, 効果の範囲, 失効と変更, 保持と削除) ' +
        'nor an API section (対象の操作, 入力, 結果, 拒否, 作用)',
    ])
  })

  it('rejects a rule that sits under no section in a feature node', () => {
    const source = [
      '# Feature: 機能',
      '',
      '## Rule: REQ-DEMO-001 規則',
      '',
      '### Example: EX-DEMO-001-01 例',
      '',
      '- When 操作する',
      '- Then 結果になる',
      '',
    ].join('\n')
    expect(verifySectionOrder(featureNode, source).map((finding) => finding.message)).toEqual([
      'REQ-DEMO-001 must sit under a section of the feature node',
    ])
  })

  it('does not ask a context root for sections', () => {
    const source = ['# Feature: 機能', '', '## Rule: REQ-DEMO-001 規則', ''].join('\n')
    expect(verifySectionOrder('docs/domain/demo/scenarios.feature.md', source)).toEqual([])
  })
})

describe('verifyFeatureNodes', () => {
  const slices = featureSlices([
    'backend/demo/trusteddevice/domain',
    'backend/demo/trusteddevice/usecases',
    'backend/demo/session/usecases',
    'backend/demo/handlers_http',
    'backend/demo/usecases',
    'backend/idmgmt/user/domain',
  ])

  it('finds a slice only where a layer sits beneath the name', () => {
    expect(slices.map((slice) => slice.path)).toEqual([
      'backend/demo/session',
      'backend/demo/trusteddevice',
      'backend/idmgmt/user',
    ])
  })

  it('matches a kebab-case node to the slice named without hyphens, through a context alias', () => {
    expect(
      verifyFeatureNodes(
        slices,
        new Set([
          'docs/domain/demo/trusted-device',
          'docs/domain/demo/session',
          'docs/domain/id-management/user',
        ]),
        { contextAliases: { idmgmt: 'id-management' }, unmappedSlices: [] },
      ),
    ).toEqual([])
  })

  it('rejects a new slice without a node, and a debt entry that no longer holds', () => {
    expect(
      verifyFeatureNodes(
        slices,
        new Set(['docs/domain/demo/trusted-device', 'docs/domain/demo/session']),
        {
          contextAliases: {},
          unmappedSlices: ['backend/demo/session'],
        },
      ).map((finding) => finding.message),
    ).toEqual([
      'backend/demo/session has a feature node now; remove it from tools/check/feature-node-debt.json',
      'backend/idmgmt/user has no feature node under docs/domain/idmgmt/',
    ])
  })
})
