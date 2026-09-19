#!/usr/bin/env bun

import { stat } from 'node:fs/promises'
import { join, normalize, resolve } from 'node:path'

/**
 * 生成サイトを HTTP で配る。Swagger UI は OpenAPI を `fetch` で読むので、`file://`
 * で開くとブラウザーがその取得を拒み、API リファレンスだけが空になる。
 * 依存を増やさないため配信は Bun の標準機能だけで行う。
 */
const root = resolve(import.meta.dir, '../../../site')
const port = Number(process.argv[2] ?? 8085)

if (!(await Bun.file(join(root, 'index.html')).exists())) {
  throw new Error(`${root} is not rendered; run mise run render-docs first`)
}

/** `..` を含む経路で site/ の外へ出られないようにしてから解決する。 */
async function resolveRequest(pathname: string): Promise<string | undefined> {
  const decoded = decodeURIComponent(pathname)
  const candidate = resolve(join(root, normalize(decoded)))
  if (candidate !== root && !candidate.startsWith(`${root}/`)) return undefined
  const found = await stat(candidate).catch(() => undefined)
  if (found?.isDirectory()) {
    const index = join(candidate, 'index.html')
    return (await Bun.file(index).exists()) ? index : undefined
  }
  return found?.isFile() ? candidate : undefined
}

const server = Bun.serve({
  port,
  async fetch(request) {
    const path = await resolveRequest(new URL(request.url).pathname)
    if (!path) return new Response('not found', { status: 404 })
    return new Response(Bun.file(path))
  },
})

console.log(`serving ${root} at http://localhost:${server.port}/`)
