import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { TOOLS_DIR } from './workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

/**
 * 正本文書として不正な本文。H1 が 2 つある。名前が許可リストに載っていれば
 * `check-specifications.ts` が落とすので、この本文を持つファイルが通ったという
 * ことは、そのファイルが検証の対象にすら入っていないということである。
 */
const INVALID_BODY = '# One\n\n# Two\n'

/** 仮の作業ツリー。正本文書の集合が閉じているかどうかだけを見る最小の形。 */
async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-workspace-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'docs', 'contexts', 'demo'), { recursive: true })
  await writeFile(join(root, 'docs', 'README.md'), '# Specification\n')
  await writeFile(join(root, 'docs', 'contexts', 'demo', 'README.md'), '# Demo\n')
  await writeFile(join(root, 'docs', 'contexts', 'demo', 'scenarios.md'), '# Demo Scenarios\n')
  return root
}

/** `--documents` を仮の作業ツリーに対して起動し、終了コードと出力を返す。 */
async function checkDocuments(root: string): Promise<{ code: number; output: string }> {
  const proc = Bun.spawn(
    ['bun', 'run', resolve(TOOLS_DIR, 'workspace/src/check-workspace.ts'), '--documents'],
    {
      cwd: TOOLS_DIR,
      env: { ...process.env, SPEC_WORKSPACE_ROOT: root },
      stdout: 'pipe',
      stderr: 'pipe',
    },
  )
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { code, output: `${stdout}${stderr}` }
}

describe('check-workspace --documents', () => {
  it('accepts a directory whose Markdown files are all canonical documents', async () => {
    const result = await checkDocuments(await workspace())
    expect(result.output).toContain('docs/contexts/demo/scenarios.md')
    expect(result.code).toBe(0)
  })

  it('rejects a Markdown file the closed set does not name', async () => {
    const root = await workspace()
    await writeFile(join(root, 'docs', 'decision.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('docs/decision.md')
  })

  it('names the canonical document a misspelled file was meant to be', async () => {
    const root = await workspace()
    await writeFile(join(root, 'docs', 'contexts', 'demo', 'scenario.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('did you mean scenarios.md')
  })

  // 名前を全部打ち間違えた作業ツリー。集めた文書が 0 件になるという形で現れるので、
  // 検査を文書の件数で条件付けていると、この最悪の場合だけ黙って通る。
  it('rejects misspelled documents even when no canonical document is found at all', async () => {
    const root = await mkdtemp(join(tmpdir(), 'check-workspace-test-'))
    cleanup.push(root)
    await mkdir(join(root, 'docs'), { recursive: true })
    await mkdir(join(root, 'work-items'), { recursive: true })
    await writeFile(join(root, 'docs', 'readme.md'), INVALID_BODY)

    const result = await checkDocuments(root)
    expect(result.code).not.toBe(0)
    expect(result.output).toContain('did you mean README.md')
  })

  it('leaves the freely named directories below the closed set alone', async () => {
    const root = await workspace()
    await mkdir(join(root, 'docs', 'runbooks'), { recursive: true })
    await mkdir(join(root, 'docs', 'development'), { recursive: true })
    await writeFile(join(root, 'docs', 'runbooks', 'anything.md'), INVALID_BODY)
    await writeFile(join(root, 'docs', 'development', 'release.md'), INVALID_BODY)

    expect((await checkDocuments(root)).code).toBe(0)
  })
})
