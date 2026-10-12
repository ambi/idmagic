import { expect, test } from 'bun:test'
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { selectChecks } from './registry.ts'
import { runChecks } from './runner.ts'

test('registry の検査は不正な操作宣言の場所と理由を報告し、有効な参照数を分けて示す', async () => {
  const root = await mkdtemp(join(tmpdir(), 'check-operation-references-'))
  try {
    for (const [path, source] of Object.entries({
      'go.mod': 'module example\n',
      'docs/README.md': '# 文書\n',
      'spec/generated/openapi/api.json': JSON.stringify({
        paths: { '/demo': { post: { operationId: 'Create' } } },
      }),
      'backend/demo/contract_test.go': `package demo
//verify:operation Create
func TestMissingColon(t *testing.T) {}
//verify:operation Create: \t
func TestMissingObservation(t *testing.T) {}
//verify:operation Unknown: 本文と状態コード
func TestUnknown(t *testing.T) {}
//verify:operation Create: 本文と状態コード
func helper() {}
//verify:operation Create: 本文と状態コード
func TestContract(t *testing.T) {}
`,
    })) {
      await mkdir(dirname(join(root, path)), { recursive: true })
      await writeFile(join(root, path), source)
    }
    const check = selectChecks(['operation-references'])
    expect(selectChecks(['all']).map((entry) => entry.name)).toContain('operation-references')
    const [result] = await runChecks(check, createWorkspaceSnapshot(root))
    expect(result?.ok).toBe(false)
    expect(result?.lines).toEqual([
      'fail  backend/demo/contract_test.go:2: missing colon after operation ID',
      'fail  backend/demo/contract_test.go:4: missing contract observation after colon',
      'fail  backend/demo/contract_test.go:6: unknown operation ID: Unknown',
      'fail  backend/demo/contract_test.go:8: directive is not directly above a Go test',
      'operation references: 1 accepted; test execution is separate evidence',
    ])
    // 同じ snapshot のキャッシュを使わず、修正した入力で過剰拒否も検出する。
    await writeFile(
      join(root, 'backend/demo/contract_test.go'),
      '//verify:operation Create: 本文と状態コード\nfunc TestContract(t *testing.T) {}\n',
    )
    const [valid] = await runChecks(check, createWorkspaceSnapshot(root))
    expect(valid).toMatchObject({
      ok: true,
      lines: ['operation references: 1 accepted; test execution is separate evidence'],
    })
    expect(valid?.operations).toBeUndefined()
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})
