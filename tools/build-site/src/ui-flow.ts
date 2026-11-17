import type { UiCatalog } from './ui-pages.ts'

type UiFlow = UiCatalog['flows'][number]

function text(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('|', '&#124;')
}

/** 長い日本語の説明で図全体が横に伸びないよう、ラベルの中で折り返す。 */
function wrapped(value: string, width = 10): string {
  const characters = Array.from(value)
  const lines: string[] = []
  for (let i = 0; i < characters.length; i += width)
    lines.push(text(characters.slice(i, i + width).join('')))
  return lines.join('<br/>')
}

function card(title: string, description: readonly string[]): string {
  return `<b>${text(title)}</b><br/><span class='flow-description'>${description.map(text).join('<br/>')}</span>`
}

/** 実際に見える内容と、それを次へ進める操作だけで利用手順を表す。 */
export function moduleFlow(flow: UiFlow, catalog: UiCatalog): string {
  const pages = new Map([...catalog.pages, ...catalog.destinations].map((page) => [page.id, page]))
  const viewIds = new Map(flow.views.map((view, index) => [view.id, `view${index}`]))
  const lines = [
    '%%{init: {"flowchart": {"padding": 12, "rankSpacing": 32, "nodeSpacing": 36}, "themeVariables": {"fontSize": "14px"}}}%%',
    `flowchart ${flow.views.length > 4 ? 'TB' : 'LR'}`,
    '  %% ui-module-flow',
    `  accTitle: ${flow.name}`,
    `  accDescr: ${flow.goal}`,
  ]
  for (const view of flow.views) {
    const page = pages.get(view.page)!
    const external = catalog.destinations.some(
      (destination) => destination.id === view.page && destination.kind === 'external',
    )
    const title = view.name ?? page.name
    lines.push(
      `  ${viewIds.get(view.id)}["${card(title, view.visible)}"]:::${external ? 'uiFlowOutside' : 'uiFlowView'}`,
    )
  }
  for (const link of flow.links) {
    lines.push(
      `  ${viewIds.get(link.from)} ${link.alternate ? '-.->' : '-->'}|"${wrapped(link.trigger, 8)}"| ${viewIds.get(link.to)}`,
    )
  }
  return lines.join('\n')
}
