import { describe, expect, it } from 'bun:test'
import { verifySubdomainClassification } from './subdomain-classification.ts'

const header = '| Specification context | Subdomain | Go package | Responsibility |'
const rule = '| --- | --- | --- | --- |'

function readme(...rows: string[]): string {
  return ['# Whole-System Specification', '', header, rule, ...rows, ''].join('\n')
}

describe('verifySubdomainClassification', () => {
  it('accepts an index that classifies every context directory', () => {
    expect(
      verifySubdomainClassification(
        readme(
          '| [Demo](contexts/demo/README.md) | Core | `backend/demo` | Demo. |',
          '| [Other](contexts/other/README.md) | Generic | `backend/other` | Other. |',
        ),
        ['demo', 'other'],
      ),
    ).toEqual([])
  })

  it('accepts every value the closed set allows', () => {
    expect(
      verifySubdomainClassification(
        readme(
          '| [A](contexts/a/README.md) | Core | `backend/a` | A. |',
          '| [B](contexts/b/README.md) | Supporting | `backend/b` | B. |',
          '| [C](contexts/c/README.md) | Generic | `backend/c` | C. |',
        ),
        ['a', 'b', 'c'],
      ),
    ).toEqual([])
  })

  it('requires no index table from a workspace that holds no context', () => {
    expect(verifySubdomainClassification('# Specification\n', [])).toEqual([])
  })

  // 列そのものが無い状態は、分類がまだ書かれていない状態である。行ごとの所見に
  // 分解すると全 Context 分の同じ所見が並ぶだけなので、表の所見 1 件で返す。
  it('rejects an index table that declares no Subdomain column', () => {
    const source = [
      '# Whole-System Specification',
      '',
      '| Specification context | Go package | Responsibility |',
      '| --- | --- | --- |',
      '| [Demo](contexts/demo/README.md) | `backend/demo` | Demo. |',
      '',
    ].join('\n')
    const findings = verifySubdomainClassification(source, ['demo'])
    expect(findings).toHaveLength(1)
    // 「Subdomain を含む」だけでは、行ごとの区分の所見（同じ語を含む）でも通って
    // しまう。列が無いことを名指しする所見であることまで固定する。
    expect(findings[0]?.message).toContain('declares no Subdomain column')
  })

  it('rejects a value outside Core, Supporting, and Generic', () => {
    const findings = verifySubdomainClassification(
      readme('| [Demo](contexts/demo/README.md) | Essential | `backend/demo` | Demo. |'),
      ['demo'],
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('Essential')
    expect(findings[0]?.message).toContain('Core, Supporting, Generic')
  })

  it('rejects an empty classification cell', () => {
    const findings = verifySubdomainClassification(
      readme('| [Demo](contexts/demo/README.md) |  | `backend/demo` | Demo. |'),
      ['demo'],
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('Core, Supporting, Generic')
  })

  // 新しい Context がディレクトリだけ先にできて索引へ載らないのが、この検査が
  // 止めたい主な失敗である。分類の欠落と同じ重さで報告する。
  it('rejects a context directory the index table does not list', () => {
    const findings = verifySubdomainClassification(
      readme('| [Demo](contexts/demo/README.md) | Core | `backend/demo` | Demo. |'),
      ['demo', 'forgotten'],
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('forgotten')
  })

  it('rejects a row naming a context directory that does not exist', () => {
    const findings = verifySubdomainClassification(
      readme(
        '| [Demo](contexts/demo/README.md) | Core | `backend/demo` | Demo. |',
        '| [Gone](contexts/gone/README.md) | Core | `backend/gone` | Gone. |',
      ),
      ['demo'],
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('gone')
  })

  it('rejects the same context listed twice', () => {
    const findings = verifySubdomainClassification(
      readme(
        '| [Demo](contexts/demo/README.md) | Core | `backend/demo` | Demo. |',
        '| [Demo again](contexts/demo/README.md) | Generic | `backend/demo` | Demo. |',
      ),
      ['demo'],
    )
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('twice')
  })

  // 件数だけを見る主張では、2 件目を読み飛ばす実装も通ってしまう。読み飛ばされた
  // 行は「索引に載っていない Context」として別の所見に化け、件数が合うためである。
  // 所見の中身まで固定して、2 件とも区分の所見であることを要求する。
  it('reports every unclassified row rather than stopping at the first', () => {
    const findings = verifySubdomainClassification(
      readme(
        '| [A](contexts/a/README.md) | Kernel | `backend/a` | A. |',
        '| [B](contexts/b/README.md) | Kernel | `backend/b` | B. |',
      ),
      ['a', 'b'],
    )
    expect(findings).toHaveLength(2)
    expect(findings.map((finding) => finding.message)).toEqual([
      expect.stringContaining('a is classified as Kernel'),
      expect.stringContaining('b is classified as Kernel'),
    ])
  })

  // 所見は README の行番号を指す。表の相対位置ではなく、開いた人が飛べる番号で
  // ないと、Context が 20 を超えた時点で探し直しになる。
  it('points at the line the offending row sits on', () => {
    const findings = verifySubdomainClassification(
      readme(
        '| [A](contexts/a/README.md) | Core | `backend/a` | A. |',
        '| [B](contexts/b/README.md) | Kernel | `backend/b` | B. |',
      ),
      ['a', 'b'],
    )
    expect(findings[0]?.line).toBe(6)
  })

  // Context Map より上には、同じ形の表が現れうる Documents 索引などがある。
  // 見出しで表を選ぶのではなくヘッダー行で選ぶことを、無関係な表を混ぜて固定する。
  it('ignores tables that are not the context index', () => {
    const source = [
      '# Whole-System Specification',
      '',
      '| File | Content |',
      '|---|---|',
      '| [glossary.md](glossary.md) | terms |',
      '',
      header,
      rule,
      '| [Demo](contexts/demo/README.md) | Core | `backend/demo` | Demo. |',
      '',
    ].join('\n')
    expect(verifySubdomainClassification(source, ['demo'])).toEqual([])
  })
})
