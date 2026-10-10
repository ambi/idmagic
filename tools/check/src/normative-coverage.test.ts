import { describe, expect, it } from 'bun:test'
import { checkNormativeCoverage, citedNormativeIds } from './normative-coverage.ts'

const declared = [
  { id: 'REQ-DEMO-001', path: 'docs/modules/demo/scenarios.feature.md' },
  { id: 'REQ-DEMO-002', path: 'docs/modules/demo/scenarios.feature.md' },
]

const file = (source: string) => [source]

describe('citedNormativeIds', () => {
  it('reads both scenario and standard ids out of a test source', () => {
    const cited = citedNormativeIds(
      [
        '//spec:covers REQ-OAUTH2-001, WCAG22-KEYBOARD: 発行を固定する。\n',
        '//spec:covers NIST63B4-PASSWORD-MINIMUM: 最短長を固定する。\n',
      ],
      ['REQ-OAUTH2-001', 'REQ-OAUTH2-002', 'WCAG22-KEYBOARD', 'NIST63B4-PASSWORD-MINIMUM'],
    )
    expect([...cited].sort()).toEqual([
      'NIST63B4-PASSWORD-MINIMUM',
      'REQ-OAUTH2-001',
      'WCAG22-KEYBOARD',
    ])
  })

  // Thirteen SAML and WS-Federation rows are named this way, and a shape that
  // expected upper-case segments could never have credited any of them.
  it('reads a mixed-case standard id', () => {
    const cited = citedNormativeIds(
      file('//spec:covers SAML2Core-BearerAssertion: 受理条件を固定する。\n'),
      ['SAML2Core-BearerAssertion', 'WSFed-PassiveSignIn'],
    )
    expect([...cited]).toEqual(['SAML2Core-BearerAssertion'])
  })

  it('does not let a longer id count as a citation of its prefix', () => {
    const cited = citedNormativeIds(file('//spec:covers REQ-DEMO-0011: 長い方だけを数える。\n'), [
      'REQ-DEMO-001',
      'REQ-DEMO-0011',
      'RFC6750-API-TOKEN-HEADER',
    ])
    expect([...cited]).toEqual(['REQ-DEMO-0011'])
  })

  // 例の住所は末尾に連番を足して作るため、隣の id を部分文字列として含む。
  // 前後どちらの端でも、別の id の一部を引用と数えてはならない。
  it('does not let a longer id count as a citation of the id it ends with', () => {
    const cited = citedNormativeIds(
      file('//spec:covers PRE-EX-DEMO-001-02: 前置きの語がある。\n'),
      ['EX-DEMO-001-01', 'EX-DEMO-001-02'],
    )
    expect([...cited]).toEqual([])
  })

  it('reads nothing when the specification declares nothing', () => {
    expect([
      ...citedNormativeIds(file('//spec:covers REQ-DEMO-001: 何かを固定する。\n'), []),
    ]).toEqual([])
  })

  // 旧形式の粗い追跡を名前だけ変えて残さないための境界。親を引用しただけの
  // テストが、その規則に属する具体例まで確認したことにはならない。
  it('does not let a rule citation cover the examples under it', () => {
    const cited = citedNormativeIds(
      file('//spec:covers REQ-OAUTH2-005: 交換の主要経路を固定する。\n'),
      ['REQ-OAUTH2-005', 'EX-OAUTH2-005-01', 'EX-OAUTH2-005-02'],
    )
    expect([...cited]).toEqual(['REQ-OAUTH2-005'])
  })

  it('credits every example a single table-driven test names', () => {
    const cited = citedNormativeIds(
      file(
        'func TestPromptNone(t *testing.T) {\n' +
          '\tcases := []struct{ id string }{\n' +
          '\t\t{id: "EX-OAUTH2-005-03"},\n' +
          '\t\t{id: "EX-OAUTH2-005-05"},\n' +
          '\t}\n}',
      ),
      ['EX-OAUTH2-005-03', 'EX-OAUTH2-005-04', 'EX-OAUTH2-005-05'],
    )
    expect([...cited].sort()).toEqual(['EX-OAUTH2-005-03', 'EX-OAUTH2-005-05'])
  })
})

describe('checkNormativeCoverage', () => {
  it('accepts a declaration a test names', () => {
    expect(
      checkNormativeCoverage({ declared, cited: new Set(['REQ-DEMO-001', 'REQ-DEMO-002']) }),
    ).toEqual([])
  })

  // 免除の一覧は無い。指摘は宣言した場所を指し、テストから名指すことだけを求める。
  // 一覧へ載せろと案内すれば、読み手を実在しない手順へ送り、そのファイルを作れば
  // 通ると誤解させる。
  it('rejects a declaration no test names, offering no list to escape into', () => {
    expect(checkNormativeCoverage({ declared, cited: new Set(['REQ-DEMO-001']) })).toEqual([
      {
        path: 'docs/modules/demo/scenarios.feature.md',
        message:
          'REQ-DEMO-002 is declared, but no test names it. ' +
          'Cite the id from the test that exercises it.',
      },
    ])
  })
})

// 正典だけを被覆と数えるかの検査。
//
// wi-567 が定めた正典は「コメント内容の先頭が id の並びで始まり、コロンで終わり、本文が
// 続く」である。散文で id に言及することを安全にするために置いた規約であり、
// wi-559 の 1 セッションで 3 回起きた取り違えは、すべてこの形の外にあった。
describe('citedNormativeIds: 正典', () => {
  const ids = ['EX-DEMO-001-01', 'EX-DEMO-001-02', 'REQ-DEMO-001']
  const cite = (source: string) => [...citedNormativeIds([source], ids)]

  it('コメント先頭の id とコロンと本文を数える', () => {
    expect(cite('//spec:covers EX-DEMO-001-01: 拒否の型を固定する。\n')).toEqual(['EX-DEMO-001-01'])
  })

  it('1 つの注記が並べた複数の id を数える', () => {
    expect(
      cite('//spec:covers EX-DEMO-001-01, EX-DEMO-001-02: 双方向を固定する。\n').sort(),
    ).toEqual(['EX-DEMO-001-01', 'EX-DEMO-001-02'])
  })

  it('読点区切りも数える', () => {
    expect(cite('//spec:covers EX-DEMO-001-01、EX-DEMO-001-02: 双方向。\n').sort()).toEqual([
      'EX-DEMO-001-01',
      'EX-DEMO-001-02',
    ])
  })

  it('ブロックコメントの継続行を数える', () => {
    expect(cite('//spec:covers EX-DEMO-001-01:\n// 拒否の型を固定する。\n')).toEqual([
      'EX-DEMO-001-01',
    ])
  })

  // 表駆動テストは id を文字列リテラルで並べる。コメントではないので前置きの
  // 問題が起きず、曖昧さも無い。
  it('コード中の文字列リテラルを数える', () => {
    expect(cite('\t\t{id: "EX-DEMO-001-01"},\n')).toEqual(['EX-DEMO-001-01'])
  })

  // ここから下が、この規約を置いた理由である。
  it('文中の言及を数えない', () => {
    expect(cite('// 台帳の EX-DEMO-001-01 は別の記録が扱う。\n')).toEqual([])
  })

  it('コロンの無いコメント先頭の id を数えない', () => {
    expect(cite('// EX-DEMO-001-01 が要求する組み立てを示す。\n')).toEqual([])
  })

  // 指示子を書いた以上、並びの中の語は書き手の意図である。前置きを禁じる規則は
  // 散文と見分けるためだけにあったので、指示子には要らない。
  it('指示子の中の前置き語は邪魔をしない', () => {
    expect(cite('//spec:covers scenario EX-DEMO-001-01: 前置きがある。\n')).toEqual([
      'EX-DEMO-001-01',
    ])
  })

  it('指示子でないコメントは、注記の形をしていても数えない', () => {
    expect(cite('// EX-DEMO-001-01: 指示子が無い。\n')).toEqual([])
  })

  // 本文は次の行から始めてもよい。現物の大半がその形で書かれていて、
  // 同一行を強いると意味の変わらない改行整形を 100 箱所に課すことになる。
  it('本文が次の行から始まる注記を数える', () => {
    expect(cite('//spec:covers EX-DEMO-001-01:\n// 拒否の型を固定する。\n')).toEqual([
      'EX-DEMO-001-01',
    ])
  })

  it('スラッシュ区切りも数える', () => {
    expect(cite('//spec:covers EX-DEMO-001-01 / EX-DEMO-001-02: 双方向。\n').sort()).toEqual([
      'EX-DEMO-001-01',
      'EX-DEMO-001-02',
    ])
  })

  it('宣言されていない id は正典の形でも数えない', () => {
    expect(cite('//spec:covers EX-DEMO-999-01: 宣言が無い。\n')).toEqual([])
  })
})

// 注記が並べる id は行をまたぐ。現物では 5 件を 2 行に折り返している箇所がある。
// 1 行に収めろと言えば折り返せなくなるので、区切りで終わる行は続きとして繋ぐ。
describe('citedNormativeIds: 行をまたぐ注記', () => {
  it('区切りで終わる行を次のコメント行へ繋いで数える', () => {
    const cited = citedNormativeIds(
      file(
        '//spec:covers REQ-DEMO-001 / EX-DEMO-001-01, EX-DEMO-001-02,\n' +
          '// EX-DEMO-001-03: 上位の権威が所有する所属は書き換えない。\n',
      ),
      ['REQ-DEMO-001', 'EX-DEMO-001-01', 'EX-DEMO-001-02', 'EX-DEMO-001-03'],
    )
    expect([...cited].sort()).toEqual([
      'EX-DEMO-001-01',
      'EX-DEMO-001-02',
      'EX-DEMO-001-03',
      'REQ-DEMO-001',
    ])
  })
})
