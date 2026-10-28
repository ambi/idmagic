import { afterEach, expect, it } from 'bun:test'
import { copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const roots: string[] = []
afterEach(() => {
  for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true })
})

function run(root: string, command: string[]): string {
  const result = Bun.spawnSync(command, { cwd: root, stderr: 'pipe' })
  if (result.exitCode !== 0) throw new Error(result.stderr.toString())
  return result.stdout.toString().trim()
}

const BASE =
  '[tasks.verify-ui]\nrun = "old-ui"\n[tasks.test-tools]\nrun = "old-tools"\n[tasks.test-go]\nrun = "old-go"'

function fixture(): string {
  const root = mkdtempSync(join(tmpdir(), 'verify-mise-'))
  roots.push(root)
  const source = join(root, 'tools/changed-packages/src')
  mkdirSync(source, { recursive: true })
  mkdirSync(join(root, 'tools/check/src'), { recursive: true })
  for (const name of ['verify', 'plan', 'gates', 'repository', 'changed-packages', 'mise-change']) {
    copyFileSync(join(import.meta.dir, `${name}.ts`), join(source, `${name}.ts`))
  }
  copyFileSync(
    join(import.meta.dir, '../../check/src/verification-tasks.ts'),
    join(root, 'tools/check/src/verification-tasks.ts'),
  )
  writeFileSync(join(root, 'go.mod'), 'module example.com/fixture\n\ngo 1.26\n')
  writeFileSync(join(root, 'app.go'), 'package fixture\n')
  writeFileSync(join(root, 'mise.toml'), BASE)
  run(root, ['git', 'init', '--quiet', '--initial-branch=main'])
  run(root, ['git', 'config', 'user.name', 'Test'])
  run(root, ['git', 'config', 'user.email', 'test@example.com'])
  run(root, ['git', 'config', 'commit.gpgsign', 'false'])
  commit(root)
  return root
}

function commit(root: string): string {
  run(root, ['git', 'add', '.'])
  run(root, ['git', 'commit', '--quiet', '-m', 'fixture'])
  return run(root, ['git', 'rev-parse', 'HEAD'])
}

function selected(root: string, ...args: string[]): string[] {
  const output = run(root, [
    process.execPath,
    'tools/changed-packages/src/verify.ts',
    '--dry-run',
    ...args,
  ])
  return [
    ...new Set(
      output
        .split('\n')
        .filter((line) => line.startsWith('run '))
        .map((line) => (line.trim().split(/\s+/)[1] ?? '').replace(':', '')),
    ),
  ]
}

it('changed は merge-base の内容と作業ツリーから層を選ぶ', () => {
  const root = fixture()
  run(root, ['git', 'switch', '--quiet', '-c', 'work'])
  const ui = BASE.replace('old-ui', 'new-ui')
  writeFileSync(join(root, 'mise.toml'), ui)
  commit(root)
  expect(selected(root, 'changed', 'main')).toEqual(['check', 'ui'])
  writeFileSync(join(root, 'mise.toml'), ui.replace('old-go', 'new-go'))
  expect(selected(root, 'changed', 'main')).toEqual(['check', 'go', 'ui'])
})

it('integration は両側の TOML を比較し、明示 revision には作業ツリーを混ぜない', () => {
  const root = fixture()
  run(root, ['git', 'switch', '--quiet', '-c', 'work'])
  const ui = BASE.replace('old-ui', 'new-ui')
  writeFileSync(join(root, 'mise.toml'), ui)
  const source = commit(root)
  run(root, ['git', 'switch', '--quiet', 'main'])
  const tools = ui.replace('old-tools', 'new-tools')
  writeFileSync(join(root, 'mise.toml'), tools)
  const result = commit(root)
  expect(selected(root, 'integration', source)).toEqual(['check'])
  writeFileSync(join(root, 'mise.toml'), tools.replace('new-ui', 'newer-ui'))
  expect(selected(root, 'integration', source)).toEqual(['check', 'ui'])
  expect(selected(root, 'integration', source, result)).toEqual(['check'])
})
