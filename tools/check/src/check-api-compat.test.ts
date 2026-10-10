import { afterEach, describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkApiCompat } from './check-api-compat.ts'
import { selectChecks } from './registry.ts'
import { runChecks } from './runner.ts'

const roots: string[] = []
afterEach(async () => {
  await Promise.all(roots.splice(0).map((root) => rm(root, { recursive: true })))
})

async function workspace(phase: unknown = 'unpublished') {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-publication-'))
  roots.push(root)
  await mkdir(join(root, 'spec', 'generated', 'openapi'), { recursive: true })
  if (phase !== undefined) {
    await writeFile(join(root, 'spec', 'release-state.json'), JSON.stringify({ phase }))
  }
  await writeFile(
    join(root, 'spec', 'idmagic.openapi.baseline.json'),
    JSON.stringify({ paths: { '/widgets': { get: { responses: { '200': {} } } } } }),
  )
  await writeFile(join(root, 'spec', 'generated', 'openapi', 'current.json'), '{"paths":{}}')
  return createWorkspaceSnapshot(root)
}

const compatibilityGates = () =>
  selectChecks(['all']).filter((check) => check.name.endsWith('api-compat'))

describe('公開状態と API 互換性ゲート', () => {
  it('未公開の API 削除を集約ゲートで拒否せず、任意の比較では検出する', async () => {
    const snapshot = await workspace()
    const results = await runChecks(compatibilityGates(), snapshot)
    expect(results).toHaveLength(1)
    expect(results[0]?.ok).toBe(true)
    expect(results[0]?.lines.join('\n')).toContain('unpublished')
    const explicit = await checkApiCompat(snapshot)
    expect(explicit.ok).toBe(false)
    expect(explicit.lines.join('\n')).toContain('/widgets: path removed')
  })

  it('公開済みの API 削除を集約ゲートで検出する', async () => {
    const results = await runChecks(compatibilityGates(), await workspace('published'))
    expect(results).toHaveLength(1)
    expect(results[0]?.ok).toBe(false)
    expect(results[0]?.lines.join('\n')).toContain('/widgets: path removed')
  })

  it('公開状態の欠落や不正値で互換性検査を免除しない', async () => {
    for (const phase of [null, true, 'publised', [], {}]) {
      const results = await runChecks(compatibilityGates(), await workspace(phase))
      expect(results[0]?.ok).toBe(false)
      expect(results[0]?.lines.join('\n')).toContain('release-state.json')
    }
    const snapshot = await workspace()
    await rm(snapshot.path('spec/release-state.json'))
    const results = await runChecks(compatibilityGates(), snapshot)
    expect(results[0]?.ok).toBe(false)
    expect(results[0]?.lines.join('\n')).toContain('release-state.json')
  })

  it('壊れた状態 JSON はファイル名を示して拒否する', async () => {
    const snapshot = await workspace()
    await writeFile(snapshot.path('spec/release-state.json'), '{')
    const results = await runChecks(compatibilityGates(), snapshot)
    expect(results[0]?.ok).toBe(false)
    expect(results[0]?.lines.join('\n')).toContain('release-state.json: invalid JSON')
  })

  it('公開後も互換な API は集約ゲートを通る', async () => {
    const snapshot = await workspace('published')
    await writeFile(
      snapshot.path('spec/generated/openapi/current.json'),
      await snapshot.read('spec/idmagic.openapi.baseline.json'),
    )
    const results = await runChecks(compatibilityGates(), snapshot)
    expect(results[0]?.ok).toBe(true)
    expect(results[0]?.lines.join('\n')).toContain('no breaking changes')
  })

  it('未公開でも現在仕様との整合とセキュリティの検査を集約から外さない', () => {
    const names = selectChecks(['all']).map((check) => check.name)
    for (const name of [
      'contract-drift',
      'status-drift',
      'security-controls',
      'event-contract',
      'work-items',
    ]) {
      expect(names).toContain(name)
    }
  })
})
