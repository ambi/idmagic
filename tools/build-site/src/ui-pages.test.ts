import { describe, expect, it } from 'bun:test'
import { createTestHost } from '@typespec/compiler/testing'
import { resolveCompilerOptions } from '@typespec/compiler'
import {
  extractUiCatalog,
  resolveRoutes,
  routeDeclarations,
  type UiRoute,
  validateUiCatalog,
} from './ui-pages.ts'
import { renderUiDocuments } from './ui-page-documents.ts'
import { renderDocumentationSite } from './render.ts'
import { Window } from 'happy-dom'

// test-input: spec/modules/frontend/models.tsp
const definitions = await Bun.file(
  new URL('../../../spec/modules/frontend/models.tsp', import.meta.url),
).text()

async function catalog(source: string) {
  const host = await createTestHost()
  host.addTypeSpecFile('models.tsp', definitions)
  host.fs.set(
    '/test/tspconfig.yaml',
    'kind: project\nentrypoint: ./main.tsp\nfeatures:\n  - auto-decorators\n',
  )
  host.addTypeSpecFile('spec/modules/identity-management/pages.tsp', `using IdMagicUi;\n${source}`)
  host.addTypeSpecFile(
    'main.tsp',
    'import "./models.tsp";\nimport "./spec/modules/identity-management/pages.tsp";',
  )
  const [options] = await resolveCompilerOptions(host.compilerHost, {
    entrypoint: '/test/main.tsp',
    cwd: '/test',
  })
  const [, diagnostics] = await host.compileAndDiagnose('main.tsp', options)
  expect(diagnostics.filter((one) => one.severity === 'error').map((one) => one.message)).toEqual(
    [],
  )
  return extractUiCatalog(host.program, '/test')
}

const example = `
model States { ready: "内容を表示する。"; submitting: "送信する。"; success: "保存した内容を表示する。"; }
model Pages {
  @page(#{id: "example", name: "例", area: Area.admin, actor: "管理者", entry: "一覧", context: "テナント", purpose: "設定を編集する", content: #["現在値"], references: #[]})
  @category(Category.users)
  example: {
    states: States;
    actions: {
      @transition(#{event: "保存", from: "ready", condition: "成功", result: "保存した内容", toState: "success", data: "入力値", mode: "data"}, Pages.example)
      save: unknown;
      @transition(#{event: "戻る"}, Pages.example)
      back: unknown;
    };
  };
}
@flow(#{id: "save", name: "設定を保存する", goal: "変更した設定を確認する", notes: #[]})
model SaveFlow {
  @view(#{state: "ready", visible: #["名前の入力"]}, Pages.example)
  edit: unknown;
  @view(#{state: "success", visible: #["保存した名前"]}, Pages.example)
  saved: unknown;
  @flowLink(#{trigger: "名前を入力して保存", action: "save"}, SaveFlow.edit, SaveFlow.saved)
  save: unknown;
}
`

const exampleRoutes = [{ id: '/example', path: '/example', markers: ['example'] }]
const resolved = async (source = example, routes: UiRoute[] = exampleRoutes) =>
  resolveRoutes(await catalog(source), routes).catalog

describe('UI ページの設計', () => {
  it('経路を実装のページマーカーから解決し、URL を手書きしない', async () => {
    const result = resolveRoutes(await catalog(example), [
      { id: '/other', path: '/other', markers: ['other'] },
      { id: '/admin/users_/$id/', path: '/admin/users/$id', markers: ['example'] },
    ])
    expect(result.findings).toEqual([])
    expect(result.catalog.pages[0]).toMatchObject({
      route: '/admin/users_/$id/',
      path: '/admin/users/$id',
    })
  })

  it('ページマーカーが一つの経路に定まらなければ、明示の route を要求する', async () => {
    const source = await catalog(example)
    expect(resolveRoutes(source, []).findings).toEqual([
      expect.stringContaining('ページマーカーに一致する経路がない'),
    ])
    const twice = [
      { id: '/a', path: '/a', markers: ['example'] },
      { id: '/b', path: '/b', markers: ['example'] },
    ]
    expect(resolveRoutes(source, twice).findings).toEqual([
      expect.stringContaining('同じページマーカーを持つ経路が複数ある'),
    ])
    const explicit = await catalog(example.replace('id: "example",', 'id: "example", route: "/b",'))
    const chosen = resolveRoutes(explicit, twice)
    expect(chosen.findings).toEqual([])
    expect(chosen.catalog.pages[0]?.path).toBe('/b')
  })

  it('明示した経路のページマーカーが別のページを指せば拒否する', async () => {
    const explicit = await catalog(example.replace('id: "example",', 'id: "example", route: "/a",'))
    expect(
      resolveRoutes(explicit, [{ id: '/a', path: '/a', markers: ['renamed'] }]).findings,
    ).toEqual([expect.stringContaining('ページ ID とページマーカーが異なる')])
    expect(resolveRoutes(explicit, [{ id: '/a', path: '/a', markers: [] }]).findings).toEqual([])
  })

  it('省略した遷移の既定値を補う', async () => {
    const page = (await resolved()).pages[0]!
    expect(page.actions[1]).toMatchObject({
      from: 'ready',
      toState: 'ready',
      condition: '常時',
      mode: 'navigation',
      data: 'なし',
      result: '移動先を表示する',
    })
  })

  it('結果を書かないページ内の更新を拒否する', async () => {
    await expect(catalog(example.replace('result: "保存した内容", ', ''))).rejects.toThrow('result')
  })

  it('詳細を主担当モジュールのナビゲーションに置き、全体のカテゴリから参照する', async () => {
    const result = await resolved()
    result.moduleNames = { 'identity-management': 'IdManagement', frontend: 'Frontend' }
    const sources = renderUiDocuments(result)
    const site = renderDocumentationSite({
      documents: [
        ...[...sources].map(([path, source]) => ({ path, source })),
        { path: 'docs/README.md', source: '# 文書\n' },
        { path: 'docs/modules/README.md', source: '# モジュール\n' },
        { path: 'docs/modules/identity-management/README.md', source: '# IdManagement\n' },
        { path: 'docs/modules/frontend/README.md', source: '# Frontend\n' },
        { path: 'docs/design/README.md', source: '# 設計\n' },
        { path: 'docs/design/application/README.md', source: '# アプリケーション\n' },
        { path: 'docs/design/application/frontend.md', source: '# フロントエンド設計\n' },
        { path: 'docs/design/application/user-interface.md', source: '# 共通の UI\n' },
      ],
      repositoryRoot: '/test',
      outputDirectory: '/test/site',
      openapiFileName: 'api.json',
      openapi: { paths: {} },
      models: [],
      moduleTags: {},
    })
    const window = new Window()
    window.document.body.innerHTML =
      site.files['docs/modules/identity-management/pages/example.html']!
    const current = window.document.querySelector('aside [aria-current="page"]')!
    expect(current.textContent).toBe('例')
    const pages = current.closest('details')!
    expect(pages.querySelector('summary')?.textContent).toBe('ページ')
    expect(pages.parentElement?.closest('details')?.querySelector('summary')?.textContent).toBe(
      'IdManagement',
    )
    expect(site.files['docs/design/application/pages/admin/users/index.html']).toContain(
      'modules/identity-management/pages/example.html',
    )
    await window.close()
  })

  it('ページの詳細に操作と結果を示し、利用手順はモジュールの一覧に置く', async () => {
    const result = await resolved()
    const documents = renderUiDocuments(result)
    expect(result.pages[0]?.module).toBe('identity-management')
    const markdown = documents.get('docs/modules/identity-management/pages/example.md')!
    expect(documents.get('docs/modules/identity-management/pages/README.md')).toContain(
      '[例](example.md)',
    )
    expect(documents.get('docs/design/application/pages/admin/users/README.md')).toContain(
      'modules/identity-management/pages/example.md',
    )
    expect([...documents.keys()].filter((path) => path.endsWith('/example.md'))).toHaveLength(1)
    expect(markdown).toContain('保存した内容')
    expect(markdown).not.toContain('```mermaid')
    expect(documents.get('docs/design/application/pages/admin/users/README.md')).not.toContain(
      '```mermaid',
    )
    expect(documents.get('docs/modules/identity-management/pages/README.md')).toContain(
      '設定を保存する',
    )
    expect(
      validateUiCatalog(result, [{ id: '/example', path: '/example', markers: ['example'] }]),
    ).toEqual([])
  })

  it('経路の欠落を双方向に検出する', async () => {
    const routes = [{ id: '/missing', path: '/missing', markers: [] }]
    const result = resolveRoutes(await catalog(example), routes)
    expect([...result.findings, ...validateUiCatalog(result.catalog, routes)]).toEqual(
      expect.arrayContaining([
        expect.stringContaining('/missing'),
        expect.stringContaining('example'),
      ]),
    )
  })

  it('途中プレビューでは未棚卸しの実装経路だけを許し、壊れた宣言は拒否する', async () => {
    const result = await resolved()
    const routes = [
      { id: '/example', path: '/example', markers: ['example'] },
      { id: '/pending', path: '/pending', markers: ['pending'] },
    ]
    expect(validateUiCatalog(result, routes, { allowUnassignedRoutes: true })).toEqual([])
    result.pages[0]!.actions[0]!.from = 'absent'
    expect(validateUiCatalog(result, routes, { allowUnassignedRoutes: true })).toEqual(
      expect.arrayContaining([expect.stringContaining('起点状態が不明')]),
    )
  })

  it('利用手順と実際の操作の到達先が違う場合や、入口から到達できない表示を拒否する', async () => {
    const result = await resolved()
    result.flows[0]!.links[0]!.to = 'edit'
    expect(
      validateUiCatalog(result, [{ id: '/example', path: '/example', markers: ['example'] }]),
    ).toEqual(
      expect.arrayContaining([
        expect.stringContaining('起点または到達先が一致しない'),
        expect.stringContaining('到達できない表示'),
      ]),
    )
  })

  it('利用手順が起点のページにない操作名を参照すれば拒否する', async () => {
    const result = await resolved(example.replace('action: "save"', 'action: "absent"'))
    expect(validateUiCatalog(result, exampleRoutes)).toEqual(
      expect.arrayContaining([expect.stringContaining('起点または到達先が一致しない')]),
    )
  })

  it('同じ ID のページと未知の起点状態を拒否する', async () => {
    const result = await resolved()
    const page = result.pages[0]!
    result.pages.push({ ...page })
    page.actions[0]!.from = 'absent'
    expect(
      validateUiCatalog(result, [{ id: '/example', path: '/example', markers: ['example'] }]),
    ).toEqual(
      expect.arrayContaining([expect.stringContaining('重複'), expect.stringContaining('absent')]),
    )
  })

  it('ページでも外部の到達先でもないプロパティへの遷移を拒否する', async () => {
    await expect(catalog(example.replace('Pages.example)', 'SaveFlow.edit)'))).rejects.toThrow(
      '遷移先',
    )
  })

  it('経路とページマーカーを AST から取得し、コメントと文字列に含む偽の宣言を数えない', () => {
    const routes = routeDeclarations(`
      // createFileRoute('/fake') <PageMarker kind="fake">
      const label = "createFileRoute('/fake2')";
      export const Route = createFileRoute('/admin/users_/$id/edit')({ component: View });
      function View() {
        return <PageMarker kind="admin-user-edit"><Form /></PageMarker>
      }
      function Dynamic({ kind }) {
        return <PageMarker kind={kind} />
      }
    `)
    expect(routes).toEqual([
      { id: '/admin/users_/$id/edit', path: '/admin/users/$id/edit', markers: ['admin-user-edit'] },
    ])
  })
})
