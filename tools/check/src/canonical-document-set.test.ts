import { describe, expect, it } from 'bun:test'
import { verifyCanonicalDocumentSet } from './canonical-document-set.ts'

describe('verifyCanonicalDocumentSet', () => {
  it('accepts a directory holding only canonical documents', () => {
    expect(
      verifyCanonicalDocumentSet([
        { directory: 'docs', files: ['README.md', 'glossary.md'] },
        { directory: 'docs/contexts/demo', files: ['README.md', 'scenarios.md'] },
      ]),
    ).toEqual([])
  })

  it('rejects a Markdown file the closed set does not name', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs', files: ['README.md', 'notes.md'] },
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.path).toBe('docs/notes.md')
  })

  // 候補提示の主張は「did you mean」まで含めて書く。候補が無いときの文言は許可名を
  // すべて並べるので、単に許可名が含まれることを見る主張は、候補提示を丸ごと削っても
  // 通ってしまう。
  it('names the canonical document a misspelled file was meant to be', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs/contexts/demo', files: ['decision.md'] },
    ])
    expect(findings[0]?.message).toContain('did you mean decisions.md?')
  })

  it('reads a name that differs only in the extension case as a misspelling', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs/contexts/demo', files: ['glossary.MD'] },
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('did you mean glossary.md?')
  })

  // 距離を測る前に両辺を小文字へそろえていることを、両側から固定する。README.md は
  // 許可名のうち唯一大文字を含むので候補側を、GLOSSARY.MD は全部が大文字なので
  // 入力側を押さえる。どちらか一方の小文字化を落とすと、対応する側が候補を失う。
  it('suggests README.md for a name written in the wrong case', () => {
    const findings = verifyCanonicalDocumentSet([{ directory: 'docs', files: ['readme.md'] }])
    expect(findings[0]?.message).toContain('did you mean README.md?')
  })

  it('suggests a canonical document for a name shouted in full uppercase', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs/contexts/demo', files: ['GLOSSARY.MD'] },
    ])
    expect(findings[0]?.message).toContain('did you mean glossary.md?')
  })

  // 候補を出す距離の上限そのものを固定する。上限が動けば、どちらかが落ちる。
  it('suggests a name two edits away', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs/contexts/demo', files: ['internl.md'] },
    ])
    expect(findings[0]?.message).toContain('did you mean internals.md?')
  })

  it('suggests nothing for a name three edits away', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs/contexts/demo', files: ['intrnl.md'] },
    ])
    expect(findings[0]?.message).not.toContain('did you mean')
  })

  it('holds each level to its own set of names', () => {
    // states.md は Context の文書であり docs/ 直下の文書ではない。逆に
    // structure.md は docs/ 直下の文書であり Context の文書ではない。
    expect(verifyCanonicalDocumentSet([{ directory: 'docs', files: ['states.md'] }])).toHaveLength(
      1,
    )
    expect(
      verifyCanonicalDocumentSet([{ directory: 'docs/contexts/demo', files: ['structure.md'] }]),
    ).toHaveLength(1)
  })

  it('falls back to the allowed names when nothing is close', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs', files: ['xyzzy-plugh-frobnitz.md'] },
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).not.toContain('did you mean')
    expect(findings[0]?.message).toContain('threat-model.md')
  })

  it('ignores files that are not Markdown', () => {
    expect(
      verifyCanonicalDocumentSet([{ directory: 'docs', files: ['diagram.svg', 'notes.txt'] }]),
    ).toEqual([])
  })

  it('reports every offending file rather than stopping at the first', () => {
    expect(
      verifyCanonicalDocumentSet([
        { directory: 'docs', files: ['one.md', 'two.md'] },
        { directory: 'docs/contexts/demo', files: ['three.md'] },
      ]),
    ).toHaveLength(3)
  })
})
