import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import viteConfig from '../vite.config'

// 経路表との照合は backend/cmd/idmagic-gateway-routes が持つ (REQ-SYSTEM-021)。その照合は
// `server.proxy` を設定の値ではなくソースの字面から読むので、字面の解釈が Vite の読み込む
// 設定と食い違うと、照合は空振りしたまま成功して見える。
//
// ここは反対側からそれを固定する。同じ規則でソースから取り出したキー集合が、Vite が実際に
// 使う設定オブジェクトのキー集合と一致することだけを確かめる。経路を手で並べないのは、
// 手で足す検査が、手で足す設定と同じ速度でずれるからである。
const proxyKeyLine = /^\s*(['"])((?:[^'"\\]|\\.)*)['"]\s*:/

function proxyKeysFromSource(source: string): string[] {
  const start = source.indexOf('proxy: {')
  expect(start).toBeGreaterThanOrEqual(0)
  const keys: string[] = []
  let depth = 0
  for (const line of source.slice(start).split('\n')) {
    const match = depth === 1 ? proxyKeyLine.exec(line) : null
    if (match?.[2] !== undefined) keys.push(match[2].replaceAll('\\\\', '\\'))
    depth += (line.match(/\{/g)?.length ?? 0) - (line.match(/\}/g)?.length ?? 0)
    if (depth <= 0) break
  }
  return keys
}

describe('development gateway', () => {
  it('exposes the same proxy contexts the source text declares', () => {
    const source = readFileSync(resolve(import.meta.dirname, '../vite.config.ts'), 'utf8')
    const fromSource = proxyKeysFromSource(source)
    expect(fromSource.length).toBeGreaterThan(0)
    expect(fromSource.sort()).toEqual(Object.keys(viteConfig.server?.proxy ?? {}).sort())
  })
})
