import { describe, expect, it } from 'bun:test'
import { checkNormativeCoverage, citedNormativeIds } from './normative-coverage.ts'

const DEBT = 'tools/check/example-coverage-debt.json'

const declared = [
  { id: 'REQ-DEMO-001', path: 'docs/contexts/demo/scenarios.feature.md' },
  { id: 'REQ-DEMO-002', path: 'docs/contexts/demo/scenarios.feature.md' },
]

const at = (...ids: string[]) =>
  new Map(ids.map((id, index) => [id, { path: 'backend/a/x_test.go', line: index + 1 }]))

const messages = (findings: Array<{ message: string }>) =>
  findings.map((finding) => finding.message)

const file = (source: string, path = 'backend/a/x_test.go') => [{ path, source }]

describe('citedNormativeIds', () => {
  it('reads both scenario and standard ids out of a test source', () => {
    const cited = citedNormativeIds(
      [
        {
          path: 'backend/a/x_test.go',
          source: '//spec:covers REQ-OAUTH2-001, WCAG22-KEYBOARD: 発行を固定する。\n',
        },
        {
          path: 'frontend/b/y.test.ts',
          source: '//spec:covers NIST63B4-PASSWORD-MINIMUM: 最短長を固定する。\n',
        },
      ],
      ['REQ-OAUTH2-001', 'REQ-OAUTH2-002', 'WCAG22-KEYBOARD', 'NIST63B4-PASSWORD-MINIMUM'],
    )
    expect([...cited.keys()].sort()).toEqual([
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
    expect([...cited.keys()]).toEqual(['SAML2Core-BearerAssertion'])
  })

  it('does not let a longer id count as a citation of its prefix', () => {
    const cited = citedNormativeIds(file('//spec:covers REQ-DEMO-0011: 長い方だけを数える。\n'), [
      'REQ-DEMO-001',
      'REQ-DEMO-0011',
      'RFC6750-API-TOKEN-HEADER',
    ])
    expect([...cited.keys()]).toEqual(['REQ-DEMO-0011'])
  })

  // 例の住所は末尾に連番を足して作るため、隣の id を部分文字列として含む。
  // 前後どちらの端でも、別の id の一部を引用と数えてはならない。
  it('does not let a longer id count as a citation of the id it ends with', () => {
    const cited = citedNormativeIds(
      file('//spec:covers PRE-EX-DEMO-001-02: 前置きの語がある。\n'),
      ['EX-DEMO-001-01', 'EX-DEMO-001-02'],
    )
    expect([...cited.keys()]).toEqual([])
  })

  it('reads nothing when the specification declares nothing', () => {
    expect([
      ...citedNormativeIds(file('//spec:covers REQ-DEMO-001: 何かを固定する。\n'), []).keys(),
    ]).toEqual([])
  })

  // 旧形式の粗い追跡を名前だけ変えて残さないための境界。親を引用しただけの
  // テストが、その規則に属する具体例まで確認したことにはならない。
  it('does not let a rule citation cover the examples under it', () => {
    const cited = citedNormativeIds(
      file('//spec:covers REQ-OAUTH2-005: 交換の主要経路を固定する。\n'),
      ['REQ-OAUTH2-005', 'EX-OAUTH2-005-01', 'EX-OAUTH2-005-02'],
    )
    expect([...cited.keys()]).toEqual(['REQ-OAUTH2-005'])
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
    expect([...cited.keys()].sort()).toEqual(['EX-OAUTH2-005-03', 'EX-OAUTH2-005-05'])
  })
})

describe('checkNormativeCoverage', () => {
  it('accepts a declaration a test names', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001', 'REQ-DEMO-002'),
      ledger: { entries: [], path: DEBT },
    })
    expect(findings).toEqual([])
  })

  it('rejects a declaration no test names and no debt entry covers', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001'),
      ledger: { entries: [], path: DEBT },
    })
    expect(findings).toHaveLength(1)
    expect(findings[0]?.path).toBe('docs/contexts/demo/scenarios.feature.md')
    expect(findings[0]?.message).toContain('REQ-DEMO-002')
    expect(findings[0]?.message).toContain('no test names it')
  })

  // 標準の側は台帳を持たない (wi-495)。台帳が無い検査は、逃げ道を案内しては
  // ならない。存在しないファイルへ載せろと言う指示は、読み手を実在しない手順へ
  // 送るうえ、そのファイルを作れば通ると誤解させる。
  it('names no escape hatch when the caller has no debt ledger', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001'),
    })
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('REQ-DEMO-002')
    expect(findings[0]?.message).toContain('no test names it')
    expect(findings[0]?.message).not.toContain('debt')
    expect(findings[0]?.message).not.toContain('list it in')
  })

  it('accepts an untested declaration the debt file carries with a reason', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001'),
      ledger: {
        entries: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(findings).toEqual([])
  })

  it('rejects a debt entry with no reason', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001'),
      ledger: { entries: [{ id: 'REQ-DEMO-002', reason: '  ' }], path: DEBT },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 is listed without a reason. State why it has no test yet.',
    ])
    expect(findings[0]?.path).toBe(DEBT)
  })

  it('rejects a debt entry that has grown a test, so the list only shrinks', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001', 'REQ-DEMO-002'),
      ledger: {
        entries: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 now has a test that names it, at backend/a/x_test.go:2. ' +
        'Remove it from the list; the list only shrinks. ' +
        'If that line only mentions the id in prose, reword it instead.',
    ])
  })

  it('rejects a debt entry nothing declares any more', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001', 'REQ-DEMO-002'),
      ledger: {
        entries: [{ id: 'REQ-DEMO-404', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-404 is listed as untested but nothing declares it any more. Remove it.',
    ])
  })

  it('rejects a debt entry listed twice', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001'),
      ledger: {
        entries: [
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
        ],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual(['REQ-DEMO-002 is listed twice. Keep one entry per id.'])
  })

  it('rejects a debt file that is not in id order', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at(),
      ledger: {
        entries: [
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
          { id: 'REQ-DEMO-001', reason: 'present when the check was introduced' },
        ],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-001 is listed after REQ-DEMO-002. Keep the list in id order so its diffs stay readable.',
    ])
  })

  // One id, one list. The refusal-shaped half of this comparison used to live
  // in security-controls.ts with its own debt file, and the two were kept
  // disjoint by reading each other. wi-490 measured what the split bought and
  // folded it back in: an id is listed here or it has a test, and nothing else
  // accounts for it.
  it('rejects the same id listed twice in one list', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: at('REQ-DEMO-001'),
      ledger: {
        entries: [
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
          { id: 'REQ-DEMO-002', reason: 'declared a refusal when the refusal check arrived' },
        ],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual(['REQ-DEMO-002 is listed twice. Keep one entry per id.'])
  })
})

// 台帳が「読んで初めて分かったこと」を保持できるかの検査。
//
// wi-538 は EX-OAUTH2-003-04 について「実装は宣言と違う型で拒否する。判断は wi-558 が
// 先に下す」と測ったが、その結論は完了した work item の散文にしか残らなかった。台帳の
// 行は他の 567 行と区別がつかないままなので、次にこの行を読む者は同じ測定をやり直す。
// blocked_by と finding はその判断を行そのものへ置く。導出できるもの (経路、パッケージ、
// 拒否かどうか) は置かない。それは spec-route が計算する。
describe('checkNormativeCoverage: 台帳が持つ判断', () => {
  const blocked = (overrides: Record<string, unknown> = {}) => ({
    declared,
    cited: at(),
    ledger: {
      entries: [
        {
          id: 'REQ-DEMO-001',
          reason: 'まだテストが無いため',
          blocked_by: 'wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets',
          finding: '製品は 401 invalid_token を返す。拒否は効いており、食い違いは型だけである。',
          ...overrides,
        },
        { id: 'REQ-DEMO-002', reason: 'まだテストが無いため' },
      ],
      path: DEBT,
    },
    knownWorkItems: new Set(['wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets']),
  })

  it('判断を持つ行を通す', () => {
    expect(checkNormativeCoverage(blocked())).toEqual([])
  })

  // 実在しない記録を指した行は、読み手をどこへも送らない。reason が空の行を落とすのと
  // 同じ理由で落とす。
  it('存在しない work item を名指した blocked_by を落とす', () => {
    expect(
      messages(checkNormativeCoverage(blocked({ blocked_by: 'wi-999-does-not-exist' }))),
    ).toEqual([
      'REQ-DEMO-001 is blocked by wi-999-does-not-exist, which is not a work item. ' +
        'Name the record that has to settle first.',
    ])
  })

  // blocked_by だけの行は「待っている」としか言わない。何を測って待つことにしたのかが
  // 無ければ、次の読み手は測り直すしかない。
  it('finding の無い blocked_by を落とす', () => {
    expect(messages(checkNormativeCoverage(blocked({ finding: undefined })))).toEqual([
      'REQ-DEMO-001 is blocked by a record without saying what was found. ' +
        'State what the implementation actually does.',
    ])
  })

  it('空の finding を落とす', () => {
    expect(messages(checkNormativeCoverage(blocked({ finding: '   ' })))).toEqual([
      'REQ-DEMO-001 has an empty finding. State what the implementation actually does.',
    ])
  })

  // 判断を持たない行は今までどおり通る。568 行のうち判断が要るのはごく一部である。
  it('判断を持たない行はそのまま通す', () => {
    expect(
      checkNormativeCoverage({
        declared,
        cited: at(),
        ledger: {
          entries: [
            { id: 'REQ-DEMO-001', reason: 'まだテストが無いため' },
            { id: 'REQ-DEMO-002', reason: 'まだテストが無いため' },
          ],
          path: DEBT,
        },
        knownWorkItems: new Set<string>(),
      }),
    ).toEqual([])
  })
})

// 名指された場所を返せるかの検査。
//
// 「テストが名指したのに台帳に残っている」という指摘は、どのファイルのどの行が名指して
// いるかを言わない。wi-559 の 1 セッションでこの指摘を 3 回受け、3 回とも rg で探すところ
// から始めた。散文で他の記録に言及しただけの行が名指しと数えられるのは照合の性質であり、
// それは変えない (測ったところ、厳しくすると正当な注記 101 件が巻き添えになる)。
// 変えるのは、どこを直せばよいかを言うかどうかである。
describe('citedNormativeIds: 名指された場所', () => {
  const sources = [
    {
      path: 'backend/a/x_test.go',
      source: 'func TestA(t *testing.T) {\n\t//spec:covers REQ-DEMO-001: 発行を固定する。\n}\n',
    },
    {
      path: 'backend/b/y_test.go',
      source: '//spec:covers REQ-DEMO-002: 拒否を固定する。\nfunc TestB(t *testing.T) {}\n',
    },
  ]

  it('id ごとに、最初に名指したファイルと行を返す', () => {
    const cited = citedNormativeIds(sources, ['REQ-DEMO-001', 'REQ-DEMO-002'])
    expect(cited.get('REQ-DEMO-001')).toEqual({ path: 'backend/a/x_test.go', line: 2 })
    expect(cited.get('REQ-DEMO-002')).toEqual({ path: 'backend/b/y_test.go', line: 1 })
  })

  // 照合そのものは 1 文字も変えない。集合として見たときの結果が変わっていないことを、
  // 場所を持たない従来の呼び出し形と同じ入力で確かめる。
  it('名指しとして数える id の集合は変わらない', () => {
    const cited = citedNormativeIds(sources, ['REQ-DEMO-001', 'REQ-DEMO-002', 'REQ-DEMO-003'])
    expect([...cited.keys()].sort()).toEqual(['REQ-DEMO-001', 'REQ-DEMO-002'])
  })
})

describe('checkNormativeCoverage: 名指された場所を指摘へ載せる', () => {
  it('台帳に残っている id について、どこが名指しているかを言う', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Map([
        ['REQ-DEMO-001', { path: 'backend/a/x_test.go', line: 9 }],
        ['REQ-DEMO-002', { path: 'backend/b/y_test.go', line: 1 }],
      ]),
      ledger: {
        entries: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 now has a test that names it, at backend/b/y_test.go:1. ' +
        'Remove it from the list; the list only shrinks. ' +
        'If that line only mentions the id in prose, reword it instead.',
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
  const cite = (source: string) => [
    ...citedNormativeIds([{ path: 'backend/a/x_test.go', source }], ids).keys(),
  ]

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
      [
        {
          path: 'backend/a/x_test.go',
          source:
            '//spec:covers REQ-DEMO-001 / EX-DEMO-001-01, EX-DEMO-001-02,\n' +
            '// EX-DEMO-001-03: 上位の権威が所有する所属は書き換えない。\n',
        },
      ],
      ['REQ-DEMO-001', 'EX-DEMO-001-01', 'EX-DEMO-001-02', 'EX-DEMO-001-03'],
    )
    expect([...cited.keys()].sort()).toEqual([
      'EX-DEMO-001-01',
      'EX-DEMO-001-02',
      'EX-DEMO-001-03',
      'REQ-DEMO-001',
    ])
  })
})
