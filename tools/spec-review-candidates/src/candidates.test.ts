import { describe, expect, it } from 'bun:test'
import {
  formatReport,
  goTestArguments,
  packagesReaching,
  parseCoverProfile,
  specCitingTests,
  unexecutedRanges,
} from './candidates.ts'

const DECLARED = ['REQ-DEMO-001', 'EX-DEMO-001-01', 'EX-DEMO-001-02', 'RFC-DEMO-ONE']

describe('specCitingTests', () => {
  it('直上の //spec:covers が宣言済みの ID を引くテストだけを返す', () => {
    const source = [
      'package demo',
      '',
      '//spec:covers EX-DEMO-001-01: 返す値',
      'func TestStart(t *testing.T) {',
      '\tstart()',
      '}',
      '',
      '// EX-DEMO-001-02 は別のテストが担当する。',
      'func TestHelperFormatsNames(t *testing.T) {',
      '\tformat()',
      '}',
      '',
      '//spec:covers EX-DEMO-999-01: 宣言されていない ID',
      'func TestUndeclared(t *testing.T) {}',
      '',
      '// 標準の行を引く。',
      '//spec:covers RFC-DEMO-ONE: 署名の検証',
      'func FuzzVerify(f *testing.F) {}',
    ].join('\n')
    expect(specCitingTests(source, DECLARED)).toEqual(['TestStart', 'FuzzVerify'])
  })

  it('本体に ID の文字列リテラルを並べる表駆動テストを返す', () => {
    const source = [
      'func TestTable(t *testing.T) {',
      '\tcases := []struct{ id string }{',
      '\t\t{id: "EX-DEMO-001-01"},',
      '\t\t{id: "EX-DEMO-001-02"},',
      '\t}',
      '\t_ = cases',
      '}',
      '',
      'func TestLogsHeader(t *testing.T) {',
      '\tt.Log("Content-Type")',
      '}',
    ].join('\n')
    expect(specCitingTests(source, DECLARED)).toEqual(['TestTable'])
  })

  it('関数の外の表と、テストでない関数の中のリテラルを、どのテストの引用にも数えない', () => {
    const source = [
      'var cases = []string{"EX-DEMO-001-01"}',
      '',
      'func helper() string { return "EX-DEMO-001-02" }',
      '',
      'func (s *suite) TestMethod(t *testing.T) {',
      '\t_ = "EX-DEMO-001-01"',
      '}',
      '',
      'func TestUsesTable(t *testing.T) {',
      '\t_ = cases',
      '}',
    ].join('\n')
    expect(specCitingTests(source, DECLARED)).toEqual([])
  })

  it('前のテストの本体にある引用を、次のテストへ持ち越さない', () => {
    const source = [
      '//spec:covers EX-DEMO-001-01: 返す値',
      'func TestFirst(t *testing.T) {',
      '\t_ = "EX-DEMO-001-02"',
      '}',
      '',
      'func TestSecond(t *testing.T) {}',
    ].join('\n')
    expect(specCitingTests(source, DECLARED)).toEqual(['TestFirst'])
  })
})

describe('packagesReaching', () => {
  const graph = [
    { importPath: 'm/backend/demo/domain', imports: [], testImports: [] },
    { importPath: 'm/backend/demo/usecases', imports: ['m/backend/demo/domain'], testImports: [] },
    { importPath: 'm/backend/http', imports: ['m/backend/demo/usecases'], testImports: [] },
    // テストだけが対象を import するパッケージも、対象のコードを実行する。
    { importPath: 'm/backend/e2e', imports: [], testImports: ['m/backend/http'] },
    { importPath: 'm/backend/other', imports: [], testImports: ['m/backend/other/support'] },
    { importPath: 'm/backend/other/support', imports: [], testImports: [] },
  ]

  it('対象のパッケージと、テストを含めて対象へ到達するパッケージを返す', () => {
    expect(packagesReaching('m/backend/demo', graph)).toEqual([
      'm/backend/demo/domain',
      'm/backend/demo/usecases',
      'm/backend/e2e',
      'm/backend/http',
    ])
  })

  it('名前が前方一致するだけの別のパッケージを対象に含めない', () => {
    expect(
      packagesReaching('m/backend/demo', [
        { importPath: 'm/backend/demonstration', imports: [], testImports: [] },
      ]),
    ).toEqual([])
  })
})

describe('goTestArguments', () => {
  it('名指したテストだけを、対象の全パッケージの被覆つきで実行する', () => {
    expect(
      goTestArguments('backend/http', ['TestStart', 'TestTable'], 'backend/demo', '/tmp/0.out'),
    ).toEqual([
      'test',
      '-run=^(TestStart|TestTable)$',
      '-coverpkg=./backend/demo/...',
      '-coverprofile=/tmp/0.out',
      './backend/http',
    ])
  })
})

describe('unexecutedRanges', () => {
  it('どのプロファイルでも実行されなかったブロックだけを、行の範囲にまとめる', () => {
    const first = parseCoverProfile(
      [
        'mode: atomic',
        'm/backend/demo/a.go:10.2,12.3 2 1',
        'm/backend/demo/a.go:14.2,16.3 2 0',
        'm/backend/demo/a.go:17.2,18.3 1 0',
        'm/backend/demo/a.go:30.2,31.3 1 0',
        'm/backend/demo/b.go:5.2,6.3 1 0',
      ].join('\n'),
    )
    const second = parseCoverProfile(
      ['mode: atomic', 'm/backend/demo/a.go:10.2,12.3 2 0', 'm/backend/demo/b.go:5.2,6.3 1 3'].join(
        '\n',
      ),
    )
    expect(unexecutedRanges([...first, ...second])).toEqual(
      new Map([
        [
          'm/backend/demo/a.go',
          [
            { start: 14, end: 18 },
            { start: 30, end: 31 },
          ],
        ],
      ]),
    )
  })
})

describe('formatReport', () => {
  const report = formatReport({
    target: 'backend/demo',
    tests: new Map([['backend/demo', ['TestStart', 'TestTable']]]),
    failedPackages: ['backend/http'],
    ranges: new Map([
      ['backend/demo/a.go', [{ start: 14, end: 18 }]],
      ['backend/demo/store/b.go', [{ start: 5, end: 5 }]],
    ]),
  })

  it('候補を仕様漏れの断定ではなく、レビューの着手点として示す', () => {
    expect(report).toContain('candidates to review')
    expect(report).toContain('not a finding')
    expect(report).not.toMatch(/is not specified|unspecified code/i)
  })

  it('位置、パッケージごとの件数、対象にしたテスト、失敗したパッケージを含める', () => {
    expect(report).toContain('backend/demo/a.go:14-18')
    expect(report).toContain('backend/demo/store/b.go:5')
    expect(report).toMatch(/backend\/demo\s+1\n/)
    expect(report).toMatch(/backend\/demo\/store\s+1\n/)
    expect(report).toContain('backend/demo TestStart')
    expect(report).toContain('backend/http')
  })
})
