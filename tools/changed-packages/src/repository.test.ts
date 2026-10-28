import { afterEach, describe, expect, it } from 'bun:test'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { readMiseChange } from './repository.ts'

const roots: string[] = []
afterEach(() => {
  for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true })
})

function git(root: string, ...args: string[]): string {
  const result = Bun.spawnSync(['git', ...args], { cwd: root, stderr: 'pipe' })
  if (result.exitCode !== 0) throw new Error(result.stderr.toString())
  return result.stdout.toString().trim()
}

function fixture(): string {
  const root = mkdtempSync(join(tmpdir(), 'mise-change-'))
  roots.push(root)
  git(root, 'init', '--quiet')
  git(root, 'config', 'user.name', 'Test')
  git(root, 'config', 'user.email', 'test@example.com')
  git(root, 'config', 'commit.gpgsign', 'false')
  writeFileSync(join(root, 'mise.toml'), '[tasks.test-go]\nrun = "old"')
  git(root, 'add', '.')
  git(root, 'commit', '--quiet', '-m', 'base')
  return root
}

describe('readMiseChange', () => {
  it('作業ツリーの内容を読み、ステージ済みの内容とは区別する', () => {
    const root = fixture()
    writeFileSync(join(root, 'mise.toml'), '[tasks.test-go]\nrun = "new"')
    git(root, 'add', '.')
    writeFileSync(join(root, 'mise.toml'), '[tasks.test-go]\nrun = "old"')
    expect(readMiseChange(root, 'HEAD').layers).toEqual([])
    writeFileSync(join(root, 'mise.toml'), '[tasks.test-go]\nrun = "newer"')
    expect(readMiseChange(root, 'HEAD').layers).toEqual(['go'])
  })

  it('指定した二つの revision を読み、作業ツリーの内容を混ぜない', () => {
    const root = fixture()
    writeFileSync(join(root, 'mise.toml'), '[tasks.test-go]\nrun = "new"')
    git(root, 'add', '.')
    git(root, 'commit', '--quiet', '-m', 'change')
    writeFileSync(join(root, 'mise.toml'), '[tools]\ngo = "changed"')
    expect(readMiseChange(root, 'HEAD~', 'HEAD')).toEqual({
      all: false,
      reasons: [],
      layers: ['go'],
    })
  })

  it('revision またはファイルを読めなければ全層を選ぶ', () => {
    const root = fixture()
    expect(readMiseChange(root, 'missing').all).toBe(true)
    expect(readMiseChange(root, 'HEAD', 'missing').all).toBe(true)
    rmSync(join(root, 'mise.toml'))
    expect(readMiseChange(root, 'HEAD').all).toBe(true)
  })
})
