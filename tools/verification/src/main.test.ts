import { expect, test } from 'bun:test'
import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const runner = resolve(import.meta.dir, 'main.ts')

test('変更操作だけに同一実行の契約テストを要求し、子のスキップと実行中の編集を拒否する', async () => {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-contract-verification-'))
  try {
    for (const directory of ['bin', 'docs', 'spec/generated/openapi', 'backend/demo/handlers_http'])
      await mkdir(join(root, directory), { recursive: true })
    await writeFile(join(root, 'go.mod'), 'module example\n')
    await writeFile(join(root, '.gitignore'), '.verification/\nbin/\nspec/generated/\n')
    await mkdir(join(root, 'backend/demo/testdata/unavailable'), { recursive: true })
    await writeFile(
      join(root, 'backend/demo/testdata/unavailable/fixture_test.go'),
      'package fixture\nimport _ "example/backend/shared/storage/testing_postgres"\n',
    )
    const handlerPath = join(root, 'backend/demo/handlers_http/handler.go')
    const handler = `package demo
func Register(g *echo.Group, d Deps) {
  g.POST("/demo", d.handleCreate)
}
func (d Deps) handleCreate(c *echo.Context) error {
  return support.NoStoreJSON(c, http.StatusOK, result)
}
`
    await writeFile(handlerPath, handler)
    await writeFile(
      join(root, 'spec/generated/openapi/api.json'),
      JSON.stringify({
        paths: {
          '/demo': {
            post: {
              operationId: 'Create',
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
    )
    for (const args of [
      ['init', '-q'],
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
    for (const scenario of ['unchanged', 'missing', 'passed', 'skipped', 'edited']) {
      await writeFile(handlerPath, scenario === 'unchanged' ? handler : `${handler}// changed\n`)
      await writeFile(
        join(root, 'backend/demo/handlers_http/contract_test.go'),
        `package demo
${['passed', 'skipped', 'edited'].includes(scenario) ? '//verify:operation Create: 本文と状態コード\n' : ''}func TestContract(t *testing.T) {
  t.Log("contract")
}
`,
      )
      const events = [
        { Action: 'pass', Package: 'example/backend/demo/handlers_http', Test: 'TestContract' },
        ...(scenario === 'skipped'
          ? [
              {
                Action: 'skip',
                Package: 'example/backend/demo/handlers_http',
                Test: 'TestContract/response',
              },
            ]
          : []),
        { Action: 'pass', Package: 'example/backend/demo/handlers_http' },
      ]
      await writeFile(
        join(root, 'bin/go'),
        `#!/bin/sh\n${scenario === 'edited' ? "printf '// during execution\\n' >> backend/demo/handlers_http/handler.go\n" : ''}printf '%s\\n' '${events.map((event) => JSON.stringify(event)).join('\n')}'\n`,
        { mode: 0o755 },
      )
      const child = Bun.spawn([process.execPath, runner, '--full', '--', './...'], {
        cwd: root,
        env: {
          ...process.env,
          PATH: `${join(root, 'bin')}:${process.env.PATH}`,
          VERIFICATION_BASE_REVISION: 'HEAD',
          VERIFICATION_REQUIRE_DB: '1',
        },
        stdout: 'pipe',
        stderr: 'pipe',
      })
      const [stdout, stderr, code] = await Promise.all([
        new Response(child.stdout).text(),
        new Response(child.stderr).text(),
        child.exited,
      ])
      expect(code, stdout + stderr).toBe(['unchanged', 'passed'].includes(scenario) ? 0 : 1)
      const result = await Bun.file(join(root, '.verification/go-tests.json')).json()
      expect(result.databases).toEqual([])
      expect(result.checks[0].operations[0].state).toBe('unverified')
      expect(result.changedOperations).toEqual(scenario === 'unchanged' ? [] : ['Create'])
      if (scenario === 'edited')
        expect(result.failures).toContain(
          'verification inputs changed during execution; rerun on a stable tree',
        )
    }
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})

test('DB スキップは既定で失敗し、明示の許可時だけ未検証として残す', async () => {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-verification-'))
  try {
    await mkdir(join(root, 'bin'))
    await mkdir(join(root, 'docs'))
    await mkdir(join(root, 'spec'))
    await mkdir(join(root, 'backend/demo/db_postgres'), { recursive: true })
    await writeFile(join(root, 'go.mod'), 'module example\n')
    await writeFile(join(root, '.gitignore'), '.verification/\nbin/\n')
    await writeFile(
      join(root, 'backend/demo/db_postgres/store_test.go'),
      `package demo
import pgtest "example/backend/shared/storage/testing_postgres"
//spec:covers REQ-DEMO-001: 保存の往復
func TestStore(t *testing.T) {
  pgtest.Require(t)
}
`,
    )
    const events = [
      {
        Action: 'output',
        Package: 'example/backend/demo/db_postgres',
        Output: 'pgtest: unavailable; skipping DB tests\n',
      },
      { Action: 'skip', Package: 'example/backend/demo/db_postgres', Test: 'TestStore' },
      { Action: 'pass', Package: 'example/backend/demo/db_postgres' },
    ]
    await writeFile(
      join(root, 'bin/go'),
      `#!/bin/sh\nprintf '%s\\n' '${events.map((event) => JSON.stringify(event)).join('\n')}'\n`,
      { mode: 0o755 },
    )
    for (const args of [
      ['init', '-q'],
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
    ]) {
      expect(Bun.spawnSync(['git', ...args], { cwd: root }).exitCode).toBe(0)
    }
    for (const scenario of [
      { name: 'default', allowSkip: '', forceRequired: '', required: true },
      { name: 'allowed', allowSkip: '1', forceRequired: '', required: false },
      { name: 'zero', allowSkip: '0', forceRequired: '', required: true },
      { name: 'true', allowSkip: 'true', forceRequired: '', required: true },
      { name: 'ci', allowSkip: '1', forceRequired: '1', required: true },
      { name: 'legacy-zero', allowSkip: '', forceRequired: '0', required: true },
    ]) {
      const report = `.verification/${scenario.name}.json`
      const child = Bun.spawn(
        [process.execPath, runner, '--report', report, '--', './backend/demo/db_postgres'],
        {
          cwd: root,
          env: {
            ...process.env,
            PATH: `${join(root, 'bin')}:${process.env.PATH}`,
            PGTEST_ALLOW_SKIP: scenario.allowSkip,
            VERIFICATION_REQUIRE_DB: scenario.forceRequired,
          },
          stdout: 'pipe',
          stderr: 'pipe',
        },
      )
      const [stdout, stderr, code] = await Promise.all([
        new Response(child.stdout).text(),
        new Response(child.stderr).text(),
        child.exited,
      ])
      expect(code, stdout + stderr).toBe(scenario.required ? 1 : 0)
      const result = await Bun.file(join(root, report)).json()
      expect(result.requireDB).toBe(scenario.required)
      expect(result.databases[0].state).toBe('unverified')
      expect(result.tests.find((entry: { name: string }) => entry.name === 'TestStore').state).toBe(
        'skipped',
      )
      expect(result.input.revision).toMatch(/^[a-f0-9]{40}$/)
      expect(result.input.fingerprint).toMatch(/^[a-f0-9]{64}$/)
      expect(result.changeResistance.state).toBe('not-collected')
    }
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})
