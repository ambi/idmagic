import { afterEach, expect, it } from 'bun:test'
import { mkdtemp, mkdir, rm, symlink, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import type { CIPlan } from '../src/ci-plan.ts'

const roots: string[] = []
const repository = resolve(import.meta.dir, '../../..')
const tasks = (
  Bun.TOML.parse(await Bun.file(join(repository, 'mise.toml')).text()) as {
    tasks: Record<string, { run: string }>
  }
).tasks
afterEach(async () => {
  await Promise.all(roots.splice(0).map((root) => rm(root, { recursive: true, force: true })))
})

function git(root: string, ...args: string[]) {
  const child = Bun.spawnSync(['git', ...args], { cwd: root })
  if (child.exitCode !== 0) throw new Error(child.stderr.toString())
  return child.stdout.toString().trim()
}

async function workspace() {
  const root = await mkdtemp(join(tmpdir(), 'ci-plan-'))
  roots.push(root)
  git(root, 'init', '-q')
  git(root, 'config', 'user.email', 'ci@example.test')
  git(root, 'config', 'user.name', 'CI')
  await mkdir(join(root, 'work-items'), { recursive: true })
  await writeFile(join(root, 'work-items/old.md'), '# 記録\n')
  git(root, 'add', '.')
  git(root, 'commit', '-qm', 'initial')
  return { root, base: git(root, 'rev-parse', 'HEAD') }
}

async function run(root: string, ...args: string[]) {
  const child = Bun.spawn(
    [process.execPath, resolve(import.meta.dir, '../src/ci-main.ts'), ...args],
    {
      cwd: root,
      env: { ...process.env, GITHUB_OUTPUT: '', GITHUB_STEP_SUMMARY: '' },
      stdout: 'pipe',
      stderr: 'pipe',
    },
  )
  const output = await new Response(child.stdout).text()
  const error = await new Response(child.stderr).text()
  expect(await child.exited, error).toBe(0)
  return JSON.parse(output) as CIPlan
}

it('CLI が二つの revision を比較し、移動元と移動先、削除、比較不能を計画へ反映する', async () => {
  const { root, base } = await workspace()
  await writeFile(join(root, 'work-items/old.md'), '# 変更\n')
  git(root, 'commit', '-qam', 'docs')
  const docs = git(root, 'rev-parse', 'HEAD')
  const plan = await run(root, 'plan', base, docs)
  expect(Object.values(plan.groups).every((group) => !group.selected)).toBe(true)
  await mkdir(join(root, 'backend'))
  git(root, 'mv', 'work-items/old.md', 'backend/example.md')
  git(root, 'commit', '-qm', 'move to backend')
  const backend = git(root, 'rev-parse', 'HEAD')
  const moved = await run(root, 'plan', docs, backend)
  expect(moved.groups.go.selected).toBe(true)
  expect(moved.groups.browser.selected).toBe(true)
  expect(moved.groups.ui.selected).toBe(false)
  expect((await run(root, 'plan', backend, docs)).groups.go.selected).toBe(true)
  git(root, 'rm', 'backend/example.md')
  git(root, 'commit', '-qm', 'delete')
  expect((await run(root, 'plan', backend, 'HEAD')).groups.go.selected).toBe(true)
  // HEAD 以外の指定した head を守り、作業ツリーの入力を混ぜない。
  expect((await run(root, 'plan', base, docs)).groups.go.selected).toBe(false)
  for (const missing of ['', '0'.repeat(40), 'missing', '--stat']) {
    const fallback = await run(root, 'plan', missing, 'HEAD')
    expect(Object.values(fallback.groups).every((group) => group.selected)).toBe(true)
  }
})

it('Git workspace がなくても全群の計画を出す', async () => {
  const root = await mkdtemp(join(tmpdir(), 'ci-plan-no-git-'))
  roots.push(root)
  const plan = await run(root, 'plan', 'missing', 'HEAD')
  expect(Object.values(plan.groups).every((group) => group.selected)).toBe(true)
})

it('mise の計画入口が GitHub 出力と summary を作り、全群と対象外の真偽値を同期する', async () => {
  const { root, base } = await workspace()
  await mkdir(join(root, 'tools'))
  await symlink(resolve(import.meta.dir, '..'), join(root, 'tools/verification'))
  const output = join(root, 'output')
  const summary = join(root, 'summary')
  for (const [path, expected] of [
    ['work-items/old.md', [false, false, false, false]],
    ['frontend/app.tsx', [false, true, true, false]],
    ['backend/tenancy/user.go', [true, false, true, false]],
    ['infra/schema/postgres.sql', [true, true, true, true]],
    ['spec/main.tsp', [true, true, true, true]],
  ] as [string, boolean[]][]) {
    await mkdir(resolve(root, path, '..'), { recursive: true })
    await writeFile(join(root, path), '# 変更\n')
    git(root, 'add', path)
    git(root, 'commit', '-qm', path)
    const head = git(root, 'rev-parse', 'HEAD')
    await writeFile(output, '')
    await writeFile(summary, '')
    const started = performance.now()
    // task の引数と実際の GITHUB_OUTPUT 形式を通す。製品テストは起動しない。
    const child = Bun.spawn(['sh', '-c', tasks['plan-ci']!.run, 'plan-ci', base, head], {
      cwd: join(root, 'tools'),
      env: { ...process.env, GITHUB_OUTPUT: output, GITHUB_STEP_SUMMARY: summary },
      stdout: 'pipe',
      stderr: 'pipe',
    })
    const stdout = await new Response(child.stdout).text()
    expect(await child.exited).toBe(0)
    const outputs = Object.fromEntries(
      (await Bun.file(output).text())
        .trim()
        .split('\n')
        .map((line) => {
          const separator = line.indexOf('=')
          return [line.slice(0, separator), line.slice(separator + 1)]
        }),
    )
    expect(JSON.parse(outputs.plan!)).toEqual(JSON.parse(stdout))
    expect(['go', 'ui', 'browser', 'schema'].map((group) => outputs[group])).toEqual(
      expected.map(String),
    )
    expect(await Bun.file(summary).text()).toContain(
      expected.some(Boolean) ? 'selected' : 'not selected',
    )
    console.info(
      JSON.stringify({
        scenario: path,
        beforeProductJobs: 5,
        afterSelectedJobs:
          Number(outputs.go === 'true') * 2 +
          ['ui', 'browser', 'schema'].filter((group) => outputs[group] === 'true').length,
        localPlanningMs: Math.round(performance.now() - started),
        hostedJobTime: 'unmeasured',
      }),
    )
    // 次の例へ直前の変更を混ぜない。
    git(root, 'revert', '--no-edit', head)
  }
})
