import { describe, expect, it } from 'bun:test'
import { moduleContextMap, moduleGroups } from './module-dependency-diagram.ts'

const HEADER =
  '| モジュール | 公開方式 | 公開パッケージ | 実装パス | 責務 |\n| --- | --- | --- | --- | --- |'
const README = [
  '# モジュール設計文書',
  '### 基盤',
  HEADER,
  '| [System](system/README.md) | 組み立て地点 | なし | `backend/cmd` | 起動 |',
  '| [Tenancy](tenancy/README.md) | `internal` | なし | `backend/tenancy` | テナント |',
  '| [Jobs](jobs/README.md) | `internal` | なし | `backend/jobs` | ジョブ |',
  '',
  '### 連携プロトコル',
  HEADER,
  '| [OAuth2](oauth2/README.md) | `internal` | なし | `backend/oauth2` | OAuth |',
  '| [Gateway](gateway/README.md) | `HTTP` | なし | `gateway` | 転送 |',
  '',
  '| 文書 | 内容 |',
  '| --- | --- |',
  '| [用語集](glossary.md) | 語 |',
].join('\n')
const relationships = (rows: string[]) =>
  [
    README,
    '',
    '| 利用元 | 利用先 | 関係の種類 | 関係 |',
    '| --- | --- | --- | --- |',
    ...rows,
  ].join('\n')

/** 生成した Markdown から、見出しの直後の Mermaid の本文を取り出す。 */
const diagram = (map: string, heading: string) => {
  const start = map.indexOf(`### ${heading}\n`)
  expect(start).toBeGreaterThanOrEqual(0)
  const body = map.slice(start).split('```mermaid\n')[1] ?? ''
  return body.slice(0, body.indexOf('```'))
}

describe('モジュールの Context Map', () => {
  it('責務表の直前の区分でまとめ、組み立て地点と別の表を除く', () => {
    expect(moduleGroups(README, 'System')).toEqual([
      { name: '基盤', modules: ['Tenancy', 'Jobs'] },
      { name: '連携プロトコル', modules: ['OAuth2', 'Gateway'] },
    ])
  })

  it('全体図は区分を節点にし、区分の間の辺を関係の種類ごとの本数で示す', () => {
    const source = relationships([
      '| OAuth2 | Tenancy | 使う | 設定を読む |',
      '| OAuth2 | Jobs | 変換 | 投入の形へ写す |',
      '| Gateway | OAuth2 | 公開契約 | 評価を要求 |',
    ])
    const map = moduleContextMap(moduleGroups(source, 'System'), [], source)
    const overview = diagram(map, '全体')

    expect(overview).toContain('g1["基盤<br/>2 モジュール"]')
    expect(overview).toContain('g2["連携プロトコル<br/>2 モジュール"]')
    expect(overview).toContain('g2 -->|"使う 1、変換 1"| g1')
    // 区分の中の辺は全体図に描かない。
    expect(overview).not.toContain('公開契約')
  })

  it('区分ごとの図は区分の外の相手を外側の節点として描き、種類で線を変える', () => {
    const source = relationships([
      '| OAuth2 | Tenancy | 使う | 設定を読む |',
      '| OAuth2 | Jobs | 変換 | 投入の形へ写す |',
      '| Gateway | OAuth2 | 公開契約 | 評価を要求 |',
    ])
    const protocols = diagram(
      moduleContextMap(moduleGroups(source, 'System'), [], source),
      '連携プロトコル',
    )

    expect(protocols).toContain('subgraph group["連携プロトコル"]')
    expect(protocols).toContain('m_Tenancy["Tenancy"]:::outside')
    expect(protocols).toContain('m_OAuth2 -->|"使う：設定を読む"| m_Tenancy')
    expect(protocols).toContain('m_OAuth2 ==>|"変換：投入の形へ写す"| m_Jobs')
    expect(protocols).toContain('m_Gateway -.->|"公開契約：評価を要求"| m_OAuth2')
  })

  it('表に宣言のない import は、提供側の型をそのまま使う関係として描く', () => {
    const map = moduleContextMap(moduleGroups(README, 'System'), [
      { sourceModule: 'OAuth2', targetModule: 'Tenancy' },
    ])

    expect(diagram(map, '連携プロトコル')).toContain('m_OAuth2 -->|"使う"| m_Tenancy')
    expect(diagram(map, '全体')).toContain('g2 -->|"使う 1"| g1')
  })

  it('境界の負債の import をラベルで区別し、台帳だけにある辺は足さない', () => {
    const map = moduleContextMap(
      moduleGroups(README, 'System'),
      [{ sourceModule: 'OAuth2', targetModule: 'Tenancy' }],
      relationships(['| OAuth2 | Tenancy | 使う | 設定を読む |']),
      [
        { sourceModule: 'OAuth2', targetModule: 'Tenancy' },
        { sourceModule: 'Tenancy', targetModule: 'OAuth2' },
      ],
    )

    expect(diagram(map, '基盤')).toContain(
      'm_OAuth2 -->|"使う：設定を読む（境界の負債）"| m_Tenancy',
    )
    expect(map).not.toContain('m_Tenancy -->')
  })

  it('責務表にないモジュール、同じ組の重複、未知の関係の種類を拒否する', () => {
    for (const rows of [
      ['| OAuth2 | Unknown | 使う | 公開 HTTP |'],
      ['| Unknown | Tenancy | 使う | 設定 |'],
      ['| OAuth2 | Tenancy | 使う | 設定 |', '| OAuth2 | Tenancy | 変換 | 通知 |'],
      ['| OAuth2 | Tenancy | 依存 | 設定 |'],
      ['| OAuth2 | Tenancy | 使う | |'],
    ]) {
      expect(() =>
        moduleContextMap(moduleGroups(README, 'System'), [], relationships(rows)),
      ).toThrow()
    }
  })

  it('公開契約の関係が import として現れたら拒否する', () => {
    expect(() =>
      moduleContextMap(
        moduleGroups(README, 'System'),
        [{ sourceModule: 'Gateway', targetModule: 'OAuth2' }],
        relationships(['| Gateway | OAuth2 | 公開契約 | 評価を要求 |']),
      ),
    ).toThrow('Gateway -> OAuth2')
  })

  it('ラベル中の引用符と山括弧を Mermaid の実体参照にする', () => {
    const source = relationships(['| OAuth2 | Tenancy | 使う | Tenant の "名前" <値> を使う |'])
    expect(moduleContextMap(moduleGroups(source, 'System'), [], source)).toContain(
      'Tenant の #quot;名前#quot; #lt;値#gt; を使う',
    )
  })
})
