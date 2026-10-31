import { describe, expect, it } from 'bun:test'
import { moduleDependencyMatrix, moduleGroups } from './module-dependency-diagram.ts'

const HEADER =
  '| モジュール | 公開方式 | 公開パッケージ | 実装パス | 責務 |\n| --- | --- | --- | --- | --- |'
const README = [
  '# モジュール設計文書',
  '### 基盤',
  HEADER,
  '| [System](system/README.md) | 組み立て地点 | なし | `backend/cmd` | 起動 |',
  '| [Tenancy](tenancy/README.md) | `internal` | なし | `backend/tenancy` | テナント |',
  '',
  '### 連携プロトコル',
  HEADER,
  '| [OAuth2](oauth2/README.md) | `internal` | なし | `backend/oauth2` | OAuth |',
  '',
  '| 文書 | 内容 |',
  '| --- | --- |',
  '| [用語集](glossary.md) | 語 |',
].join('\n')
const relationships = (rows: string[]) =>
  [README, '', '| 利用元 | 利用先 | 関係 |', '| --- | --- | --- |', ...rows].join('\n')

describe('モジュール間の依存行列', () => {
  it('責務表の直前の区分でまとめ、組み立て地点と別の表を除く', () => {
    expect(moduleGroups(README, 'System')).toEqual([
      { name: '基盤', modules: ['Tenancy'] },
      { name: '連携プロトコル', modules: ['OAuth2'] },
    ])
  })

  it('区分の順に並べ、利用元の行と利用先の列の交点に import を示す', () => {
    expect(
      moduleDependencyMatrix(moduleGroups(README, 'System'), [
        { sourceModule: 'OAuth2', targetModule: 'Tenancy' },
      ]).split('\n'),
    ).toEqual([
      '| 利用元＼利用先 | 1 | 2 |',
      '| --- | --- | --- |',
      '| 1 Tenancy | — | — |',
      '| 2 OAuth2 | [●](../../modules/README.md#モジュールの間の依存 "OAuth2 → Tenancy：import") | — |',
    ])
  })

  it('関係の表のラベルを保ち、import にない HTTP とポートの関係も描く', () => {
    const source = relationships([
      '| OAuth2 | Tenancy | Tenant を使う |',
      '| Tenancy | OAuth2 | 設定の変更を通知 |',
    ])
    const diagram = moduleDependencyMatrix(
      moduleGroups(source, 'System'),
      [{ sourceModule: 'OAuth2', targetModule: 'Tenancy' }],
      source,
    )
    expect(diagram).toContain(
      '[●](../../modules/README.md#モジュールの間の依存 "OAuth2 → Tenancy：import、Tenant を使う")',
    )
    expect(diagram).toContain(
      '[◇](../../modules/README.md#モジュールの間の依存 "Tenancy → OAuth2：HTTP またはポート、設定の変更を通知")',
    )
    expect(diagram.match(/\[●\]/g)).toHaveLength(1)
  })

  it('負債の import を記号とラベルで示し、台帳だけにある辺は足さない', () => {
    const diagram = moduleDependencyMatrix(
      moduleGroups(README, 'System'),
      [{ sourceModule: 'OAuth2', targetModule: 'Tenancy' }],
      relationships(['| OAuth2 | Tenancy | Tenant を使う |']),
      [
        { sourceModule: 'OAuth2', targetModule: 'Tenancy' },
        { sourceModule: 'Tenancy', targetModule: 'OAuth2' },
      ],
    )
    expect(diagram).toContain(
      '[!](../../modules/README.md#モジュールの間の依存 "OAuth2 → Tenancy：境界の負債、Tenant を使う")',
    )
    expect(diagram).toContain('| 1 Tenancy | — | — |')
  })

  it('ラベルのない負債の import も識別する', () => {
    const edge = { sourceModule: 'OAuth2', targetModule: 'Tenancy' }
    expect(moduleDependencyMatrix(moduleGroups(README, 'System'), [edge], '', [edge])).toContain(
      '[!](../../modules/README.md#モジュールの間の依存 "OAuth2 → Tenancy：境界の負債")',
    )
  })

  it('責務表にないモジュールと同じ組の重複した関係を拒否する', () => {
    for (const rows of [
      ['| OAuth2 | Unknown | 公開 HTTP |'],
      ['| Unknown | Tenancy | 設定 |'],
      ['| OAuth2 | Tenancy | 設定 |', '| OAuth2 | Tenancy | 通知 |'],
    ]) {
      expect(() =>
        moduleDependencyMatrix(moduleGroups(README, 'System'), [], relationships(rows)),
      ).toThrow()
    }
  })

  it('関係ラベル中の引用符と HTML をリンクタイトルのエンティティにする', () => {
    const source = relationships(['| OAuth2 | Tenancy | Tenant の "名前" <値> を使う |'])
    expect(moduleDependencyMatrix(moduleGroups(README, 'System'), [], source)).toContain(
      'Tenant の &quot;名前&quot; &lt;値&gt; を使う',
    )
  })
})
