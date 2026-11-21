import {
  compile,
  formatDiagnostic,
  getAutoDecoratorTargets,
  getAutoDecoratorValue,
  getDoc,
  getSourceLocation,
  getTypeName,
  NodeHost,
  resolveCompilerOptions,
  type Model,
  type EnumValue,
  type ModelProperty,
  type Operation,
  type Program,
} from '@typespec/compiler'
import { relative, resolve } from 'node:path'

type PageInfo = {
  id: string
  name: string
  area: string
  route?: string
  actor: string
  entry: string
  context: string
  purpose: string
  content: string[]
  references: string[]
}
type FieldInfo = {
  label: string
  control: string
  initial: string
  condition: string
  /** 束縛先のプロパティ名と同じ値を送る入力では省く。 */
  mapping?: string
  repeated?: boolean
}
type UiField = FieldInfo & { id: string; type: string; binding?: string }
type UiForm = {
  id: string
  name: string
  condition: string
  commit: string
  fields: UiField[]
}
type UiAction = {
  id: string
  event: string
  from: string
  condition: string
  result: string
  toState: string
  data: string
  mode: string
  destination: string
  operations: string[]
}
type UiDestination = {
  id: string
  name: string
  kind: 'external' | 'shared'
  description: string
}
type RouteNote = { route: string; kind: string; reason: string; destination?: string }
type UiPage = PageInfo & {
  /** resolveRoutes が実装の経路から埋めるまでは空文字列。 */
  route: string
  path: string
  module: string
  category: string
  categoryName: string
  source: string
  forms: UiForm[]
  parameters: UiField[]
  states: Record<string, string>
  actions: UiAction[]
}
export type UiCatalog = {
  pages: UiPage[]
  destinations: UiDestination[]
  routeNotes: RouteNote[]
  moduleNames: Record<string, string>
  flows: Array<{
    id: string
    name: string
    goal: string
    notes: string[]
    module: string
    views: Array<{ id: string; page: string; state: string; name?: string; visible: string[] }>
    links: Array<{
      from: string
      to: string
      trigger: string
      alternate?: boolean
      action?: { page: string; id: string }
    }>
  }>
}
/** markers は経路のファイルが静的に描画する PageMarker の kind。 */
export type UiRoute = { id: string; path: string; markers: string[] }

/** UI の宣言は HTTP のコンパイルへ混ぜず、型の参照だけを共有する。 */
export async function loadUiCatalog(root: string): Promise<UiCatalog> {
  const entrypoint = resolve(root, 'spec/modules/frontend/main.tsp')
  const [options, diagnostics] = await resolveCompilerOptions(NodeHost, {
    entrypoint,
    cwd: root,
  })
  if (diagnostics.length)
    throw new Error(diagnostics.map((one) => formatDiagnostic(one)).join('\n'))
  const program = await compile(NodeHost, entrypoint, { ...options, noEmit: true })
  if (program.hasError())
    throw new Error(program.diagnostics.map((one) => formatDiagnostic(one)).join('\n'))
  const catalog = extractUiCatalog(program, root)
  for (const module of new Set(['frontend', ...catalog.pages.map((page) => page.module)])) {
    const readme = await Bun.file(resolve(root, `docs/modules/${module}/README.md`)).text()
    const title = readme.match(/^# (.+)$/m)?.[1]
    if (!title) throw new Error(`${module}: モジュールのタイトルがない`)
    catalog.moduleNames[module] = title
  }
  return catalog
}

/** compiler が解決した参照を、文書生成と照合で使う値へ変換する。 */
export function extractUiCatalog(program: Program, root: string): UiCatalog {
  const metadata = <T>(name: string, target: ModelProperty | Model): T | undefined =>
    getAutoDecoratorValue(program, `IdMagicUi.${name}`, target)?.info as T | undefined
  const properties = (model: Model, name: string): ModelProperty[] => {
    const type = model.properties.get(name)?.type
    if (type?.kind !== 'Model') throw new Error(`${getTypeName(model)}: ${name} がモデルではない`)
    return [...type.properties.values()]
  }
  const pageTargets = [
    ...getAutoDecoratorTargets(program, 'IdMagicUi.page').keys(),
  ] as ModelProperty[]
  const destinationTargets = [
    ...getAutoDecoratorTargets(program, 'IdMagicUi.destination').keys(),
  ] as ModelProperty[]
  pageTargets.sort((left, right) => {
    const a = getSourceLocation(left)
    const b = getSourceLocation(right)
    return a.file.path.localeCompare(b.file.path) || a.pos - b.pos
  })
  const identifiers = new Map<ModelProperty, string>()
  for (const target of pageTargets) identifiers.set(target, metadata<PageInfo>('page', target)!.id)
  for (const target of destinationTargets)
    identifiers.set(target, metadata<UiDestination>('destination', target)!.id)
  const fields = (model: Model, prefix: string): UiField[] =>
    [...model.properties.values()].map((property) => {
      const id = `${prefix}/${property.name}`
      const info = metadata<Partial<FieldInfo> & Pick<FieldInfo, 'label' | 'control'>>(
        'field',
        property,
      )
      if (!info) throw new Error(`${id}: 入力項目の説明がない`)
      const binding = getAutoDecoratorValue(program, 'IdMagicUi.bind', property)?.source as
        | ModelProperty
        | undefined
      if (!binding && !info.mapping)
        throw new Error(`${id}: API プロパティへ束縛しない入力には mapping が必要`)
      // 束縛した入力は型を書き写さず、束縛先から導く。
      const type =
        binding && property.type.kind === 'Intrinsic' && property.type.name === 'unknown'
          ? binding.type
          : property.type
      return {
        initial: '空',
        condition: '常時',
        ...info,
        id,
        type: getTypeName(type),
        binding: binding ? `${getTypeName(binding.model!)}.${binding.name}` : undefined,
      }
    })
  const pages = pageTargets.map((target): UiPage => {
    const info = metadata<PageInfo>('page', target)!
    const source = relative(root, getSourceLocation(target).file.path).replaceAll('\\', '/')
    const module = source.match(/^spec\/modules\/([^/]+)\/[^/]+\.tsp$/)?.[1]
    if (!module) throw new Error(`${info.id}: ページの主担当を宣言元の配置から解決できない`)
    const category = getAutoDecoratorValue(program, 'IdMagicUi.category', target)?.category as
      | EnumValue
      | undefined
    if (!category) throw new Error(`${info.id}: UI のカテゴリがない`)
    if (target.type.kind !== 'Model') throw new Error(`${info.id}: ページがモデルではない`)
    const model = target.type
    const forms = properties(model, 'forms').map((property): UiForm => {
      const form = metadata<Omit<UiForm, 'id' | 'fields' | 'condition'> & { condition?: string }>(
        'form',
        property,
      )
      if (!form || property.type.kind !== 'Model')
        throw new Error(`${info.id}/${property.name}: フォームの説明がない`)
      return {
        condition: '常時',
        ...form,
        id: property.name,
        fields: fields(property.type, `${info.id}/${property.name}`),
      }
    })
    const parameterModel = model.properties.get('parameters')?.type
    if (parameterModel?.kind !== 'Model') throw new Error(`${info.id}: parameters がない`)
    const states = Object.fromEntries(
      properties(model, 'states').map((property) => {
        if (property.type.kind !== 'String')
          throw new Error(`${info.id}: 状態の説明が文字列ではない`)
        return [property.name, property.type.value]
      }),
    )
    const actions = properties(model, 'actions').map((property): UiAction => {
      const transition = getAutoDecoratorValue(program, 'IdMagicUi.transition', property)
      const destination = identifiers.get(transition?.destination as ModelProperty)
      if (!transition || !destination) throw new Error(`${info.id}/${property.name}: 遷移先が不明`)
      const operations = getAutoDecoratorValue(program, 'IdMagicUi.uses', property)?.operations as
        | Operation[]
        | undefined
      const declared = transition.info as Partial<UiAction> & Pick<UiAction, 'event'>
      if (declared.mode === 'data' && !declared.result)
        throw new Error(`${info.id}/${property.name}: ページ内の更新には result が必要`)
      return {
        from: 'ready',
        toState: 'ready',
        condition: '常時',
        mode: 'navigation',
        data: 'なし',
        result: '移動先を表示する',
        ...declared,
        id: property.name,
        destination,
        operations: (operations ?? []).map((operation) => getTypeName(operation)),
      }
    })
    return {
      ...info,
      route: info.route ?? '',
      path: '',
      module,
      category: String(category.value.value ?? category.value.name),
      categoryName: getDoc(program, category.value) ?? category.value.name,
      // enum の value は compiler によって EnumValue として渡される。
      area:
        typeof info.area === 'string'
          ? info.area
          : (info.area as { value: { name: string } }).value.name,
      source,
      forms,
      parameters: fields(parameterModel, `${info.id}/parameters`),
      states,
      actions,
    }
  })
  const routeNotes = [...getAutoDecoratorTargets(program, 'IdMagicUi.route').keys()].map(
    (target) => metadata<RouteNote>('route', target as ModelProperty)!,
  )
  const flows = [...getAutoDecoratorTargets(program, 'IdMagicUi.flow').keys()].map((target) => {
    if (target.kind !== 'Model') throw new Error('利用手順がモデルではない')
    const info = metadata<Omit<UiCatalog['flows'][number], 'module' | 'views' | 'links'>>(
      'flow',
      target,
    )!
    const source = relative(root, getSourceLocation(target).file.path).replaceAll('\\', '/')
    const module = source.match(/^spec\/modules\/([^/]+)\//)?.[1]
    if (!module) throw new Error(`${info.id}: 利用手順の主担当がない`)
    const views: UiCatalog['flows'][number]['views'] = []
    const links: UiCatalog['flows'][number]['links'] = []
    for (const property of target.properties.values()) {
      const view = getAutoDecoratorValue(program, 'IdMagicUi.view', property)
      if (view) {
        const page = identifiers.get(view.page as ModelProperty)
        if (!page) throw new Error(`${info.id}/${property.name}: 表示するページが不明`)
        views.push({
          ...(view.info as { state: string; name?: string; visible: string[] }),
          id: property.name,
          page,
        })
      }
      const edge = getAutoDecoratorValue(program, 'IdMagicUi.flowLink', property)
      if (edge) {
        const from = edge.from as ModelProperty
        const to = edge.to as ModelProperty
        if (from.model !== target || to.model !== target)
          throw new Error(`${info.id}: 別の利用手順の表示をつないでいる`)
        const { action, ...link } = edge.info as {
          trigger: string
          alternate?: boolean
          action?: string
        }
        // 操作を持つページは、すべての表示を集めた後で起点の表示から決める。
        links.push({
          ...link,
          from: from.name,
          to: to.name,
          action: action ? { page: '', id: action } : undefined,
        })
      }
    }
    for (const link of links) {
      if (!link.action) continue
      const page = views.find((view) => view.id === link.from)?.page
      if (!page) throw new Error(`${info.id}: 操作 ${link.action.id} の起点の表示が不明`)
      link.action.page = page
    }
    return { ...info, module, views, links }
  })
  return {
    pages,
    flows,
    moduleNames: {},
    destinations: destinationTargets.map(
      (target) => metadata<UiDestination>('destination', target)!,
    ),
    routeNotes,
  }
}

const pageMarkerRule = `
id: page-marker
language: tsx
rule:
  kind: string_fragment
  inside:
    kind: string
    inside:
      kind: jsx_attribute
      has: { kind: property_identifier, regex: ^kind$ }
      inside:
        any: [{ kind: jsx_opening_element }, { kind: jsx_self_closing_element }]
        has: { field: name, regex: ^PageMarker$ }
`

/**
 * 宣言のページ ID と実装の PageMarker を対応させ、経路と URL を埋めた写しを返す。
 * route を明示したページは、その経路のマーカーが別の ID を指していないことだけを確かめる。
 */
export function resolveRoutes(
  catalog: UiCatalog,
  routes: readonly UiRoute[],
): { catalog: UiCatalog; findings: string[] } {
  const findings: string[] = []
  const pages = catalog.pages.map((page) => {
    const candidates = page.route
      ? routes.filter((route) => route.id === page.route)
      : routes.filter((route) => route.markers.includes(page.id))
    if (page.route && candidates.length === 0)
      findings.push(`${page.id}: 明示した経路に実装がない: ${page.route}`)
    else if (candidates.length === 0)
      findings.push(
        `${page.id}: ページマーカーに一致する経路がない。ID を PageMarker の kind に合わせる`,
      )
    else if (candidates.length > 1)
      findings.push(
        `${page.id}: 同じページマーカーを持つ経路が複数ある。route で選ぶ: ${candidates.map((one) => one.id).join(', ')}`,
      )
    const route = candidates.length === 1 ? candidates[0]! : undefined
    if (route?.markers.length && !route.markers.includes(page.id))
      findings.push(
        `${page.id}: ページ ID とページマーカーが異なる: ${route.id} (${route.markers.join(', ')})`,
      )
    return route ? { ...page, route: route.id, path: route.path } : page
  })
  return { catalog: { ...catalog, pages }, findings }
}

/** ファイル名や JSX の見た目を推測せず、正式な経路宣言を読む。 */
export function routeDeclarations(source: string): UiRoute[] {
  // kind を式で決めるマーカーは数えず、その経路のページは route を明示する。
  const marked = Bun.spawnSync(
    ['ast-grep', 'scan', '--stdin', '--json=compact', '--inline-rules', pageMarkerRule],
    { stdin: new TextEncoder().encode(source) },
  )
  if (marked.exitCode !== 0) throw new Error(marked.stderr.toString())
  const markers = [
    ...new Set<string>(
      JSON.parse(marked.stdout.toString() || '[]').map((match: { text: string }) => match.text),
    ),
  ]
  const matches = Bun.spawnSync(
    ['ast-grep', 'run', '--lang', 'tsx', '-p', 'createFileRoute($ID)', '--json=compact', '--stdin'],
    { stdin: new TextEncoder().encode(source) },
  )
  if (matches.exitCode !== 0 && matches.exitCode !== 1) throw new Error(matches.stderr.toString())
  const routes: UiRoute[] = JSON.parse(matches.stdout.toString() || '[]').map(
    (match: { metaVariables: { single: { ID: { text: string } } } }) => {
      const text = match.metaVariables.single.ID.text
      if (
        !(
          (text.startsWith("'") && text.endsWith("'")) ||
          (text.startsWith('"') && text.endsWith('"'))
        )
      )
        throw new Error('経路 ID が文字列リテラルではない')
      const id = text.slice(1, -1)
      const path =
        id
          .split('/')
          .map((part) => (part.endsWith('_') ? part.slice(0, -1) : part))
          .join('/')
          .replace(/\/$/, '') || '/'
      return { id, path, markers }
    },
  )
  const root = Bun.spawnSync(
    [
      'ast-grep',
      'run',
      '--lang',
      'tsx',
      '-p',
      'createRootRoute($$$ARGS)',
      '--json=compact',
      '--stdin',
    ],
    { stdin: new TextEncoder().encode(source) },
  )
  if (root.exitCode === 0) {
    routes.push({ id: '__root__', path: '/', markers: [] })
  }
  return routes
}

export function validateUiCatalog(
  catalog: UiCatalog,
  routes: readonly UiRoute[],
  options: { allowUnassignedRoutes?: boolean } = {},
): string[] {
  const findings: string[] = []
  const ids = new Set<string>()
  const paths = new Set<string>()
  const pages = new Map(catalog.pages.map((page) => [page.id, page]))
  const destinations = new Set(catalog.destinations.map((one) => one.id))
  const assigned = new Map<string, string>()
  for (const item of [...catalog.pages, ...catalog.destinations]) {
    if (ids.has(item.id)) findings.push(`識別子の重複: ${item.id}`)
    ids.add(item.id)
  }
  for (const page of catalog.pages) {
    // 経路を解決できなかったページは resolveRoutes が報告する。
    if (page.path) {
      if (paths.has(page.path)) findings.push(`ページ URL の重複: ${page.path}`)
      paths.add(page.path)
      if (assigned.has(page.route)) findings.push(`経路の重複: ${page.route}`)
      assigned.set(page.route, page.path)
    }
    if (!page.states.ready) findings.push(`${page.id}: ready 状態がない`)
    if (page.actions.length === 0) findings.push(`${page.id}: 操作と結果の記述がない`)
    for (const action of page.actions) {
      if (!page.states[action.from])
        findings.push(`${page.id}/${action.id}: 起点状態が不明: ${action.from}`)
      const target = pages.get(action.destination)
      if (target && !target.states[action.toState])
        findings.push(`${page.id}/${action.id}: 到達状態が不明: ${action.toState}`)
      if (!target && !destinations.has(action.destination))
        findings.push(`${page.id}/${action.id}: 遷移先が不明: ${action.destination}`)
    }
  }
  for (const note of catalog.routeNotes) {
    if (assigned.has(note.route)) findings.push(`経路の重複: ${note.route}`)
    assigned.set(note.route, '')
    if (!note.reason.trim()) findings.push(`経路の除外理由がない: ${note.route}`)
    if (!routes.some((route) => route.id === note.route))
      findings.push(`説明の経路に実装がない: ${note.route}`)
  }
  for (const route of routes) {
    if (!assigned.has(route.id) && !options.allowUnassignedRoutes)
      findings.push(`実装の経路が設計にない: ${route.id}`)
  }
  const flowIds = new Set<string>()
  for (const flow of catalog.flows) {
    const id = `${flow.module}/${flow.id}`
    if (flowIds.has(id)) findings.push(`利用手順の識別子の重複: ${id}`)
    flowIds.add(id)
    const views = new Map(flow.views.map((view) => [view.id, view]))
    if (views.size < 2 || !flow.links.length)
      findings.push(`${id}: 利用手順の表示と遷移が不足している`)
    for (const view of flow.views) {
      const page = pages.get(view.page)
      if (page && !page.states[view.state])
        findings.push(`${id}/${view.id}: 表示の状態が不明: ${view.state}`)
      if (!page && !destinations.has(view.page))
        findings.push(`${id}/${view.id}: 表示するページが不明`)
      if (!view.visible.length) findings.push(`${id}/${view.id}: 利用者に見える内容がない`)
    }
    const reached = new Set(flow.views.length ? [flow.views[0]!.id] : [])
    for (const edge of flow.links) {
      const from = views.get(edge.from)
      const to = views.get(edge.to)
      if (!from || !to) {
        findings.push(`${id}: 遷移が表示を参照していない`)
        continue
      }
      const action = edge.action
        ? pages.get(edge.action.page)?.actions.find((one) => one.id === edge.action!.id)
        : undefined
      if (
        edge.action &&
        (!action ||
          edge.action.page !== from.page ||
          action.destination !== to.page ||
          action.from !== from.state ||
          action.toState !== to.state)
      )
        findings.push(`${id}: 利用手順と参照した操作の起点または到達先が一致しない`)
      if (!edge.action && pages.has(from.page))
        findings.push(`${id}: ページ内の操作を参照していない`)
    }
    for (let i = 0; i < views.size; i++)
      for (const edge of flow.links) if (reached.has(edge.from)) reached.add(edge.to)
    if (flow.views.some((view) => !reached.has(view.id)))
      findings.push(`${id}: 入口から到達できない表示がある`)
  }
  return findings
}
