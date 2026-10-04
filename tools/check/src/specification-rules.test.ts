import { describe, expect, it } from 'bun:test'
import {
  codeSlices,
  goDeclarations,
  verifyEarsStatements,
  verifyFeatureSliceSpecifications,
  verifyRuleFields,
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

const requirement = (...body: string[]): string =>
  [
    '## 操作',
    '',
    '### 管理者による無効化',
    '',
    '#### REQ-DEMO-001 無効化は記録を残す',
    '',
    ...body,
  ].join('\n')
const earsMessages = (...body: string[]) =>
  verifyEarsStatements('docs/domain/demo/user/README.md', requirement(...body), 'Demo').map(
    (finding) => finding.message,
  )

describe('verifyEarsStatements', () => {
  it('accepts every pattern, a compound preamble, and several responses', () => {
    expect(
      earsMessages(
        '- Demo は、ユーザー名を前後の空白を除いて保存する。',
        '- User が `Active` 以外の間、Demo は、その User の新規のサインインを拒否する。',
        '- 発行者がロールを持つ間、要求されたとき、Demo は、操作を実行する。',
        '- User がロックされている間、Demo は、サインインを拒否する。',
        '- 規則が評価できない間、Demo は、所属を与えない。',
        '- 管理者が User を無効化したとき、Demo は、`UserDisabled` を発行する。',
        '- `a`、`b`、`c` のいずれかが空の場合、Demo は、422 と `invalid_request` で拒否し、User を変えない。',
        '- 再利用を禁止するテナントでは、Demo は、履歴にあるパスワードへの変更を拒否する。',
        '- 監査を有効にしたテナントでは、User が `Disabled` の間、管理者が再開を要求したとき、Demo は、User を `Active` にする。',
        '- **判断**：理由の欄は構文を問わない場合がある。',
        '| 条件 | 結果 |',
        '| --- | --- |',
        '| 空の場合 | 拒否 |',
      ),
    ).toEqual([])
  })

  it('rejects a statement without the responder', () => {
    expect(earsMessages('- User のユーザー名は、前後の空白を除いて保存する。')).toEqual([
      'REQ-DEMO-001 statement names no responder; write 「Demo は、」 after the preamble',
    ])
  })

  it('rejects two sentences on one line', () => {
    expect(earsMessages('- 無効化したとき、Demo は、記録する。記録は消さない。')).toEqual([
      'REQ-DEMO-001 statement holds more than one sentence; write one requirement per line',
    ])
  })

  it('rejects a responder written twice', () => {
    expect(earsMessages('- 無効化したとき、Demo は、記録し、Demo は、通知する。')).toEqual([
      'REQ-DEMO-001 statement names the responder 「Demo は、」 more than once',
    ])
  })

  it('does not read 間 after a kanji noun as a state marker', () => {
    expect(earsMessages('- 猶予期間、Demo は、記録する。')).toEqual([
      'REQ-DEMO-001 preamble 「猶予期間」 ends with no marker; end it with では、の間、とき、or 場合、',
    ])
  })

  it('rejects a preamble clause that ends without a marker', () => {
    expect(earsMessages('- 無効化の要求に対して、Demo は、記録する。')).toEqual([
      'REQ-DEMO-001 preamble 「無効化の要求に対して」 ends with no marker; end it with では、の間、とき、or 場合、',
    ])
  })

  it('rejects clauses out of order and a second trigger', () => {
    expect(
      earsMessages(
        '- 無効化したとき、`Active` の間、Demo は、記録する。',
        '- 無効化したとき、名前が空の場合、Demo は、拒否する。',
      ),
    ).toEqual([
      'REQ-DEMO-001 preamble puts a state (の間) after a trigger; order them 構成、状態、契機 or 場合',
      'REQ-DEMO-001 preamble has more than one trigger (とき or 場合)',
    ])
  })

  it('reserves 場合 for unwanted behaviour', () => {
    expect(earsMessages('- 名前が空の場合を除いた状態の間、Demo は、記録する。')).toEqual([
      'REQ-DEMO-001 uses 「場合」 outside the unwanted-behaviour marker',
    ])
  })

  it('rejects a condition inside the response', () => {
    expect(
      earsMessages(
        '- 要求されたとき、Demo は、スコープを含む場合だけ実行する。',
        '- 要求されたとき、Demo は、有効な限り実行する。',
        '- 要求されたとき、Demo は、ロールを持つ間だけ実行する。',
        '- 起動したとき、Demo は、テナントがなければ作る。',
      ),
    ).toEqual([
      'REQ-DEMO-001 response contains the condition 「場合」; move it into the preamble',
      'REQ-DEMO-001 response contains the condition 「限り」; move it into the preamble',
      'REQ-DEMO-001 response contains the condition 「の間」; move it into the preamble',
      'REQ-DEMO-001 response contains the condition 「〜ば」; move it into the preamble',
    ])
  })

  it('rejects ability, permission, recommendation, and definition endings', () => {
    expect(
      earsMessages(
        '- Demo は、トークンを失効できる。',
        '- Demo は、説明を省いてもよい。',
        '- Demo は、上限を表示することが望ましい。',
        '- Demo は、有効期限を 30 日とする。',
        '- Demo は、一意である。',
      ),
    ).toEqual([
      'REQ-DEMO-001 response ends with 「できる」; state what the responder does',
      'REQ-DEMO-001 response ends with 「てもよい」; state what the responder does',
      'REQ-DEMO-001 response ends with 「望ましい」; state what the responder does',
      'REQ-DEMO-001 response ends with 「とする」; state what the responder does',
      'REQ-DEMO-001 response ends with 「である」; state what the responder does',
    ])
  })

  it('ignores code spans, superseded requirements, and lines outside requirements', () => {
    const source = [
      '## 操作',
      '',
      '- 操作の節の説明は要件文ではない。',
      '',
      '#### REQ-DEMO-001 古い要件 (superseded by REQ-DEMO-002)',
      '',
      '- 後継に置き換えた。',
      '',
      '#### REQ-DEMO-002 新しい要件',
      '',
      '- `場合` を含む名前を指定された場合、Demo は、`invalid_name` で拒否する。',
    ].join('\n')
    expect(verifyEarsStatements('docs/domain/demo/user/README.md', source, 'Demo')).toEqual([])
  })
})

describe('verifyFeatureSliceSpecifications', () => {
  const slices = codeSlices([
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
      verifyFeatureSliceSpecifications(
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
      verifyFeatureSliceSpecifications(
        slices,
        new Set(['docs/domain/demo/trusted-device', 'docs/domain/demo/session']),
        {
          contextAliases: {},
          unmappedSlices: ['backend/demo/session'],
        },
      ).map((finding) => finding.message),
    ).toEqual([
      'backend/demo/session has a feature slice specification now; remove it from tools/check/feature-slice-debt.json',
      'backend/idmgmt/user has no feature slice specification under docs/domain/idmgmt/',
    ])
  })
})
