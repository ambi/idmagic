import { describe, expect, it } from 'bun:test'
import { TERMINOLOGY_RULES, type TerminologyRule, verifyTerminology } from './terminology.ts'

const rule = (term: string): TerminologyRule => {
  const found = TERMINOLOGY_RULES.find((candidate) => candidate.term === term)
  if (!found) throw new Error(`no rule for ${term}`)
  return found
}

describe('用語検査', () => {
  it('採らない表記を、行と採用語つきで指摘する', () => {
    const findings = verifyTerminology([
      { file: 'docs/architecture/deployment.md', source: '# 概要\n\n配備の不変条件。\n' },
    ])

    expect(findings).toEqual([
      {
        file: 'docs/architecture/deployment.md',
        line: 3,
        column: 1,
        term: '配備',
        message:
          '「配備」は採らない表記。「デプロイ」（行為）または「デプロイメント」（ビュー名） を使う。',
      },
    ])
  })

  it('採用語だけの文書は通す', () => {
    expect(
      verifyTerminology([
        {
          file: 'docs/architecture/deployment.md',
          source: 'デプロイメントアーキテクチャは共通トポロジーを持つ。\n',
        },
      ]),
    ).toEqual([])
  })

  // 共起で通すのは別概念に限る。private key は起動時シークレットではない。
  it('残すと決めた共起は通し、同じ行の別概念は通さない', () => {
    const findings = verifyTerminology([
      {
        file: 'docs/design/security/secrets.md',
        source: '秘密鍵と秘密情報は残すが、秘密は残さない。\n',
      },
    ])

    expect(findings.map((finding) => finding.column)).toEqual([14])
    expect(findings).toHaveLength(1)
  })

  // 「トポロジー」は「トポロジ」を含む。共起の許可が occurrence を覆う位置で
  // 判定されないと、採用した表記そのものが毎回落ちる。
  it('採用語が採らない表記を含む場合でも、採用語を落とさない', () => {
    expect(
      verifyTerminology([{ file: 'docs/architecture/deployment.md', source: '共通トポロジー\n' }]),
    ).toEqual([])
    expect(
      verifyTerminology([{ file: 'docs/architecture/deployment.md', source: '共通トポロジ\n' }]),
    ).toHaveLength(1)
  })

  // 「参照」は「〜を参照する」と読めるため、構成の名前には使わない。
  // 動詞としての「参照」まで落とすと、文書間の案内が書けなくなる。
  it('構成の名前としての「参照」だけを対象にする', () => {
    const findings = verifyTerminology([
      {
        file: 'docs/architecture/deployment.md',
        source: '参照トポロジーと参照プロファイルは、別の文書を参照する。\n',
      },
    ])

    expect(findings.map((finding) => finding.term)).toEqual(['参照トポロジー', '参照プロファイル'])
  })

  it('既定、コンピュート、資材を、採用語へ寄せる', () => {
    const findings = verifyTerminology([
      { file: 'docs/design/infrastructure/platform.md', source: '既定のコンピュートと資材。\n' },
    ])

    expect(findings.map((finding) => finding.term)).toEqual(['既定', 'コンピュート', '資材'])
  })

  // 「コンピューティング」は「コンピュート」を含まないので、許可の literal は要らない。
  it('採用語のコンピューティングを落とさない', () => {
    expect(
      verifyTerminology([
        {
          file: 'docs/design/infrastructure/platform.md',
          source: 'コンピューティングとデフォルト。\n',
        },
      ]),
    ).toEqual([])
  })

  // 許可の literal より前に現れた occurrence を、後続の literal が覆ったことに
  // してはならない。1 行に両方あるときだけ現れる誤りである。
  it('許可 literal の外にある同じ語を、位置で区別する', () => {
    const findings = verifyTerminology([
      { file: 'docs/design/data/lifecycle.md', source: '容量は保存容量とは別である。\n' },
    ])

    expect(findings.map((finding) => finding.column)).toEqual([1])
  })

  it('1 行に複数回現れた語を、すべて指摘する', () => {
    const findings = verifyTerminology([{ file: 'docs/README.md', source: '配備と配備。\n' }])

    expect(findings.map((finding) => finding.column)).toEqual([1, 4])
  })

  it('採らない表記ごとに採用語を持つ', () => {
    for (const candidate of TERMINOLOGY_RULES) {
      expect(candidate.term.length).toBeGreaterThan(0)
      expect(candidate.adopt.length).toBeGreaterThan(0)
      for (const allowed of candidate.allow ?? []) {
        // 対象語を含まない literal は、どの occurrence も覆わない。書いた側は
        // 免除したつもりでいるので、規則表の側で落とす。
        expect(allowed.literal).toContain(candidate.term)
        expect(allowed.reason.length).toBeGreaterThan(0)
      }
    }
  })

  it('副詞的な「実行時」ではなくビュー名だけを対象にする', () => {
    expect(
      verifyTerminology([
        { file: 'docs/contexts/jobs/internals.md', source: '実行時に評価し、実行時刻へ達する。\n' },
      ]),
    ).toEqual([])
    expect(rule('実行時アーキテクチャ').term).toBe('実行時アーキテクチャ')
  })

  it('普通名詞としての規則を通し、文書名としての規則だけを対象にする', () => {
    expect(
      verifyTerminology([
        {
          file: 'docs/contexts/authorization/internals.md',
          source: '認可規則と検証規則は残す。\n',
        },
      ]),
    ).toEqual([])
    expect(
      verifyTerminology([
        { file: 'docs/design/application/README.md', source: '[設計規則](design-guidelines.md)\n' },
      ]),
    ).toHaveLength(1)
  })
})
