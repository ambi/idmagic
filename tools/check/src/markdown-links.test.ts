import { describe, expect, it } from 'bun:test'
import { verifyMarkdownLinks } from './markdown-links.ts'

describe('verifyMarkdownLinks', () => {
  it('reports missing relative targets and anchors', () => {
    const documents = new Map([
      [
        'docs/guide.md',
        [
          '# Guide',
          '[target](target.md#repeated-1)',
          '[missing target](missing.md)',
          '[missing anchor](target.md#absent)',
          '[local anchor](#guide)',
          '[directory](../tools/)',
          '[external](https://example.com/missing)',
          '[mail](mailto:docs@example.com)',
        ].join('\n'),
      ],
      ['docs/target.md', ['# Repeated!', '# Repeated!', '<a id="explicit-anchor"></a>'].join('\n')],
    ])
    const existingPaths = new Set(['docs', 'docs/guide.md', 'docs/target.md', 'tools'])

    expect(verifyMarkdownLinks(documents, existingPaths)).toEqual([
      {
        file: 'docs/guide.md',
        line: 2,
        target: 'missing.md',
        message: 'relative link target does not exist: docs/missing.md',
      },
      {
        file: 'docs/guide.md',
        line: 2,
        target: 'target.md#absent',
        message: 'Markdown anchor does not exist in docs/target.md: #absent',
      },
    ])
  })

  it('ignores link examples inside nested and tilde code fences', () => {
    const source = [
      '````markdown',
      '[outer example](missing.md)',
      '```markdown',
      '[nested example](also-missing.md)',
      '```',
      '````',
      '~~~text',
      '[tilde example](third-missing.md)',
      '~~~',
      '[real](missing.md)',
    ].join('\n')

    expect(verifyMarkdownLinks(new Map([['guide.md', source]]), new Set(['guide.md']))).toEqual([
      {
        file: 'guide.md',
        line: 10,
        target: 'missing.md',
        message: 'relative link target does not exist: missing.md',
      },
    ])
  })

  it('resolves reference links and percent-encoded paths', () => {
    const documents = new Map([
      [
        'README.md',
        [
          '[guide][docs]',
          '[missing][gone]',
          '',
          '[docs]: docs/My%20Guide.md#overview',
          '[gone]: absent.md',
        ].join('\n'),
      ],
      ['docs/My Guide.md', '# Overview'],
    ])
    const existingPaths = new Set(['README.md', 'docs', 'docs/My Guide.md'])

    expect(verifyMarkdownLinks(documents, existingPaths)).toEqual([
      {
        file: 'README.md',
        line: 1,
        target: 'absent.md',
        message: 'relative link target does not exist: absent.md',
      },
    ])
  })

  it('reports local file URLs instead of silently exempting them', () => {
    expect(
      verifyMarkdownLinks(
        new Map([['history.md', '[machine-local](file:///Users/example/ARCHITECTURE.md)']]),
        new Set(['history.md']),
      ),
    ).toEqual([
      {
        file: 'history.md',
        line: 1,
        target: 'file:///Users/example/ARCHITECTURE.md',
        message: 'local file URL is not portable',
      },
    ])
  })

  it('uses the full syntax tree for multiline links and code spans', () => {
    const source = [
      '`code',
      '[ignored](inside-code.md)',
      '`',
      '',
      '[multiline',
      'link](missing.md)',
    ].join('\n')

    expect(verifyMarkdownLinks(new Map([['guide.md', source]]), new Set(['guide.md']))).toEqual([
      {
        file: 'guide.md',
        line: 5,
        target: 'missing.md',
        message: 'relative link target does not exist: missing.md',
      },
    ])
  })

  it('checks unused reference definitions', () => {
    expect(
      verifyMarkdownLinks(new Map([['guide.md', '[unused]: missing.md\n']]), new Set(['guide.md'])),
    ).toEqual([
      {
        file: 'guide.md',
        line: 1,
        target: 'missing.md',
        message: 'relative link target does not exist: missing.md',
      },
    ])
  })

  it('avoids heading slug collisions and recognizes explicit HTML anchors', () => {
    const source = [
      '# Repeated',
      '# Repeated-1',
      '# Repeated',
      '<span id="explicit"></span>',
      '<a name=legacy></a>',
      '[duplicate](#repeated-2)',
      '[explicit](#explicit)',
      '[legacy](#legacy)',
    ].join('\n')

    expect(verifyMarkdownLinks(new Map([['guide.md', source]]), new Set(['guide.md']))).toEqual([])
  })

  it('fails closed when no Markdown documents were collected', () => {
    expect(verifyMarkdownLinks(new Map(), new Set())).toEqual([
      {
        file: '.',
        line: 1,
        target: '',
        message: 'no Markdown documents were collected',
      },
    ])
  })
})
