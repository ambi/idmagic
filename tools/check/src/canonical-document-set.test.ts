import { describe, expect, it } from 'bun:test'
import { verifyCanonicalDocumentSet } from './canonical-document-set.ts'

/** モジュールの印。印のないモジュールの段には、どの文書も置けない。 */
const marker = { directory: 'docs/modules/demo', files: ['README.md'] }

describe('verifyCanonicalDocumentSet', () => {
  it('accepts a directory holding only canonical documents', () => {
    expect(
      verifyCanonicalDocumentSet([
        { directory: 'docs', files: ['README.md'] },
        {
          directory: 'docs/modules',
          files: ['README.md', 'glossary.md', 'scenarios.feature.md'],
        },
        { directory: 'docs/requirements', files: ['README.md', 'quality.md'] },
        { directory: 'docs/design/infrastructure', files: ['README.md', 'network.md'] },
        marker,
        { directory: 'docs/modules/demo', files: ['README.md', 'glossary.md'] },
      ]),
    ).toEqual([])
  })

  it('rejects the shared glossary and scenarios at their former requirements location', () => {
    const findings = verifyCanonicalDocumentSet([
      {
        directory: 'docs/requirements',
        files: ['README.md', 'glossary.md', 'scenarios.feature.md'],
      },
    ])
    expect(findings.map((finding) => finding.path)).toEqual([
      'docs/requirements/glossary.md',
      'docs/requirements/scenarios.feature.md',
    ])
  })

  it('accepts design areas and records beside the module overview', () => {
    expect(
      verifyCanonicalDocumentSet([
        {
          directory: 'docs/modules/demo',
          files: ['README.md', 'data.md', 'observability.md', 'decisions.md', 'risks.md'],
        },
        {
          directory: 'docs/modules/demo/features',
          files: ['user.md', 'user.lifecycle.md', 'user.feature.md'],
        },
      ]),
    ).toEqual([])
  })

  it('keeps shared vocabulary out of the features directory', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/modules/demo/features', files: ['user.md', 'glossary.md'] },
    ])
    expect(findings.map((finding) => finding.path)).toEqual([
      'docs/modules/demo/features/glossary.md',
    ])
  })

  it('rejects any document below the features directory or in another module directory', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/modules/demo/features/user', files: ['README.md'] },
      { directory: 'docs/modules/demo/design', files: ['architecture.md'] },
    ])
    expect(findings.map((finding) => finding.message)).toEqual([
      'not a canonical document; a module holds only its own documents and features/',
      'not a canonical document; a module holds only its own documents and features/',
    ])
  })

  it('admits no document in a module without README.md', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs/modules/demo', files: ['glossary.md', 'scenarios.feature.md'] },
      { directory: 'docs/modules/demo/features', files: ['user.md'] },
    ])
    expect(findings.map((finding) => finding.path)).toEqual([
      'docs/modules/demo/glossary.md',
      'docs/modules/demo/scenarios.feature.md',
      'docs/modules/demo/features/user.md',
    ])
  })

  it('rejects a free-named document beside the fixed module documents', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/modules/demo', files: ['csv-transfer.md'] },
    ])
    expect(findings.map((finding) => finding.path)).toEqual(['docs/modules/demo/csv-transfer.md'])
  })

  it('rejects the file kinds of the former layout in a module', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      {
        directory: 'docs/modules/demo',
        files: ['states.md', 'internals.md', 'scenarios.feature.md', 'design.md'],
      },
    ])
    expect(findings).toHaveLength(4)
  })

  it('rejects the feature directory layout and the former appendix names', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      {
        directory: 'docs/modules/demo/features',
        files: ['README.md', 'acceptance.feature.md', 'user.examples.feature.md'],
      },
    ])
    expect(findings.map((finding) => finding.path)).toEqual([
      'docs/modules/demo/features/README.md',
      'docs/modules/demo/features/user.examples.feature.md',
    ])
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
      marker,
      { directory: 'docs/modules/demo', files: ['qualty.md'] },
    ])
    expect(findings[0]?.message).toContain('did you mean quality.md?')
  })

  it('reads a name that differs only in the extension case as a misspelling', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/modules/demo', files: ['glossary.MD'] },
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
      marker,
      { directory: 'docs/modules/demo', files: ['GLOSSARY.MD'] },
    ])
    expect(findings[0]?.message).toContain('did you mean glossary.md?')
  })

  // 候補を出す距離の上限そのものを固定する。上限が動けば、どちらかが落ちる。
  it('suggests a name two edits away', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/modules/demo', files: ['standrd.md'] },
    ])
    expect(findings[0]?.message).toContain('did you mean standards.md?')
  })

  it('suggests nothing for a name three edits away', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/requirements', files: ['qualityxyz.md'] },
    ])
    expect(findings[0]?.message).not.toContain('did you mean')
  })

  it('rejects a separate design document for a feature', () => {
    const findings = verifyCanonicalDocumentSet([
      marker,
      { directory: 'docs/modules/demo/features', files: ['task.md', 'task.design.md'] },
    ])
    expect(findings.map((finding) => finding.path)).toEqual([
      'docs/modules/demo/features/task.design.md',
    ])
  })

  it('holds each level to its own set of names', () => {
    // quality.md はモジュールの文書であり docs/ 直下の文書でも機能でもない。
    expect(verifyCanonicalDocumentSet([{ directory: 'docs', files: ['quality.md'] }])).toHaveLength(
      1,
    )
    expect(
      verifyCanonicalDocumentSet([
        marker,
        { directory: 'docs/modules/demo/features', files: ['quality.md'] },
      ]),
    ).toHaveLength(1)
    expect(
      verifyCanonicalDocumentSet([{ directory: 'docs/design/security', files: ['network.md'] }]),
    ).toHaveLength(1)
  })

  it('falls back to the allowed names when nothing is close', () => {
    const findings = verifyCanonicalDocumentSet([
      { directory: 'docs', files: ['xyzzy-plugh-frobnitz.md'] },
    ])
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).not.toContain('did you mean')
    expect(findings[0]?.message).toContain('README.md')
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
        marker,
        { directory: 'docs/modules/demo', files: ['Three.md'] },
      ]),
    ).toHaveLength(3)
  })
})
