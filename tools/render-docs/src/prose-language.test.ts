import { describe, expect, it } from 'bun:test'
import { ALLOWED_ENGLISH, findEnglishProse } from './prose-language.ts'

describe('findEnglishProse', () => {
  it('本文に残った英文をページと文字列で報告する', () => {
    const pages = {
      'models/user.html':
        '<main><h1>User</h1><p>The user who signs in to the tenant.</p><p>テナントへサインインする User。</p></main>',
    }

    expect(findEnglishProse(pages)).toEqual([
      { page: 'models/user.html', text: 'The user who signs in to the tenant.' },
    ])
  })

  // 文中のリンクや強調を境界にすると、英文が 3 語未満の断片に切れて見逃される。
  it('リンクや強調を挟んだ英文を一つの文として検出する', () => {
    const pages = {
      'a.html': '<p>The <a href="#u">user</a> who <em>signs</em> in to the tenant.</p>',
    }

    expect(findEnglishProse(pages).map((finding) => finding.text)).toEqual([
      'The user who signs in to the tenant.',
    ])
  })

  it('日本語の文に埋め込んだ英文も検出する', () => {
    const pages = { 'a.html': '<p>この値は、the value is ignored when unset、として扱う。</p>' }

    expect(findEnglishProse(pages).map((finding) => finding.text)).toEqual([
      'the value is ignored when unset',
    ])
  })

  // 機能語を条件にしないと、規格名、用語、識別子の並びまで英文として報告する。
  it('機能語を含まない英単語の並びは英文としない', () => {
    const pages = {
      'a.html':
        '<p>OAuth 2.0 Token Exchange と Problem Details を使い、Security Event Token を送る。</p>',
    }

    expect(findEnglishProse(pages)).toEqual([])
  })

  it('コード、整形済みテキスト、スクリプト、スタイル、SVG の中は読まない', () => {
    const pages = {
      'a.html': [
        '<p>値は <code>when the flag is set</code> で決まる。</p>',
        '<pre>flowchart LR\n  A[the start of it] --> B</pre>',
        '<script>const note = "this is not prose"</script>',
        '<style>/* the rule for the page */</style>',
        '<svg><text>the label of an edge</text></svg>',
      ].join(''),
    }

    expect(findEnglishProse(pages)).toEqual([])
  })

  // 表のセルを区切らずにつなぐと、隣り合う識別子の列が一つの英文に見える。
  it('要素の境界をまたいで単語をつながない', () => {
    const pages = {
      'a.html': '<table><tr><td>excluded</td><td>MAY</td><td>to</td><td>the</td></tr></table>',
    }

    expect(findEnglishProse(pages)).toEqual([])
  })

  // Gherkin のステップは英語のキーワードで始まり、日本語の文が続く。キーワードだけで
  // 英文とすると、受け入れの具体例のすべてのステップを誤検出する。
  it('Gherkin のステップのキーワードに続く英語の名前は英文としない', () => {
    const pages = {
      'a.html':
        '<li>When WS-Trust Issue の RST を受信する</li><li>When the token is expired, it is refused</li>',
    }

    expect(findEnglishProse(pages).map((finding) => finding.text)).toEqual([
      'When the token is expired, it is refused',
    ])
  })

  it('許容語の表にある規格名と文書名は英文としない', () => {
    const pages = {
      'a.html':
        '<h2>Proof Key for Code Exchange by OAuth Public Clients</h2><p>出典は <em>The Art of Readable Code</em> である。</p>',
    }

    expect(findEnglishProse(pages)).toEqual([])
  })

  it('HTML 以外の生成物は読まない', () => {
    expect(findEnglishProse({ 'openapi/idmagic.json': '"The user who signs in."' })).toEqual([])
  })

  it('許容語にはそれぞれ許容する理由の分類がある', () => {
    for (const entry of ALLOWED_ENGLISH) {
      expect(['規格名', '文書名', '製品名', '記法', '用語']).toContain(entry.reason)
      expect(entry.phrase.trim()).toBe(entry.phrase)
    }
  })
})
