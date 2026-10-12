import { expect, test } from 'bun:test'
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'

test('CLI は予定するファイルと Git 差分から参照不足を保存し、従来の棚卸しを保つ', async () => {
  const root = await mkdtemp(join(tmpdir(), 'contract-evidence-preflight-'))
  const runner = resolve(import.meta.dir, 'contract-evidence-inventory.ts')
  try {
    for (const [path, source] of Object.entries({
      'go.mod': 'module example\n',
      '.gitignore': '.verification/\nspec/generated/\n',
      'docs/README.md': '# 文書\n',
      'spec/generated/openapi/api.json': JSON.stringify({
        paths: {
          '/demo': {
            get: {
              operationId: 'ReadDemo',
              responses: {
                '200': {
                  content: {
                    'application/json': {
                      schema: { type: 'object', properties: { id: { type: 'string' } } },
                    },
                  },
                },
              },
            },
          },
        },
      }),
      'backend/demo/handlers_http/routes.go': `package demo
func Register(g *echo.Group) { g.GET("/demo", handleReadDemo) }
func handleReadDemo(c *echo.Context) error { return c.JSON(http.StatusOK, value) }
`,
    })) {
      await mkdir(dirname(join(root, path)), { recursive: true })
      await writeFile(join(root, path), source)
    }
    for (const args of [
      ['init', '-q', '-b', 'main'],
      ['add', '.'],
      [
        '-c',
        'user.name=fixture',
        '-c',
        'user.email=fixture@example.invalid',
        'commit',
        '-qm',
        'fixture',
      ],
    ])
      expect(Bun.spawnSync(['git', ...args], { cwd: root }).exitCode).toBe(0)
    const run = async (args: string[]) => {
      const child = Bun.spawn([process.execPath, runner, ...args], {
        cwd: root,
        stdout: 'pipe',
        stderr: 'pipe',
      })
      const [stdout, stderr, code] = await Promise.all([
        new Response(child.stdout).text(),
        new Response(child.stderr).text(),
        child.exited,
      ])
      expect(code, stdout + stderr).toBe(0)
    }
    await run(['--changed', '--base-revision', 'HEAD', '--target', 'backend/shared/new/value.go'])
    const report = join(root, '.verification/contract-evidence-preflight.json')
    expect(await Bun.file(report).exists()).toBe(true)
    expect(await Bun.file(report).json()).toMatchObject({
      baseRevision: 'HEAD',
      changedFiles: [],
      targets: ['backend/shared/new/value.go'],
      missingReferences: ['ReadDemo'],
      execution: 'not-collected',
    })
    await mkdir(join(root, 'backend/shared/new'), { recursive: true })
    await writeFile(join(root, 'backend/shared/new/value.go'), 'package new\n')
    await writeFile(
      join(root, 'backend/demo/handlers_http/read_test.go'),
      '//verify:operation ReadDemo: 本文と状態コード\nfunc TestRead(t *testing.T) {}\n',
    )
    await run(['--changed', '--base-revision', 'HEAD'])
    const declared = await Bun.file(report).json()
    expect(declared.changedFiles).toContain('backend/shared/new/value.go')
    expect(declared.missingReferences).toEqual([])
    expect(declared.operations[0].references).toEqual([
      { path: 'backend/demo/handlers_http/read_test.go', name: 'TestRead' },
    ])
    expect(declared.execution).toBe('not-collected')
    await run([])
    const inventory = await Bun.file(
      join(root, '.verification/contract-evidence-inventory.json'),
    ).json()
    expect(Array.isArray(inventory)).toBe(true)
    expect(inventory[0].operationId).toBe('ReadDemo')
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})
