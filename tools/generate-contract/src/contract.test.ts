import { afterEach, describe, expect, it } from 'bun:test'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { collectOperations, renderContract } from './contract.ts'

// OPENAPI31-OPERATION-ID: 生成文書の operationId は文書全体で一意であり、
// 重複を生成器が黙って捨ててはならない。

const duplicateDocument = {
  paths: {
    '/default': { get: { operationId: 'ReadThing' } },
    '/named/{id}': { post: { operationId: 'ReadThing' } },
  },
}

describe('generated contract operation collection', () => {
  it('carries deprecation dates from the published operation to the runtime table', () => {
    const document = {
      paths: {
        '/old': {
          get: {
            operationId: 'ExportAdminAuditEvents',
            deprecated: true,
            'x-deprecated-since': '2026-10-25',
            'x-sunset-at': '2027-10-25',
          },
        },
      },
    }
    expect(renderContract(collectOperations(document))).toContain(
      '"ExportAdminAuditEvents": {Since: "2026-10-25", Sunset: "2027-10-25"}',
    )
    expect(() =>
      collectOperations({
        paths: { '/old': { get: { ...document.paths['/old'].get, 'x-sunset-at': '2027-10-24' } } },
      }),
    ).toThrow('invalid deprecation dates')
  })
  it('rejects duplicate operation IDs and reports both routes', () => {
    // OPENAPI31-OPERATION-ID: 同じ operationId を持つ両経路を示して拒否する。
    expect(() => collectOperations(duplicateDocument)).toThrow(
      'duplicate operationId ReadThing: GET /default and POST /named/{id}',
    )
  })
})

describe('generated contract CLI', () => {
  const directories: string[] = []
  afterEach(async () => {
    await Promise.all(directories.splice(0).map((directory) => rm(directory, { recursive: true })))
  })

  it('generated contract entry point rejects duplicate operation IDs', async () => {
    const directory = await mkdtemp(resolve(tmpdir(), 'idmagic-generate-contract-'))
    directories.push(directory)
    const input = resolve(directory, 'openapi.json')
    const output = resolve(directory, 'operations_gen.go')
    await writeFile(input, JSON.stringify(duplicateDocument), 'utf8')

    const process = Bun.spawn(
      [
        Bun.which('bun') ?? 'bun',
        'run',
        resolve(import.meta.dir, 'main.ts'),
        '--input',
        input,
        '--output',
        output,
      ],
      { stderr: 'pipe' },
    )
    const [exitCode, stderr] = await Promise.all([
      process.exited,
      new Response(process.stderr).text(),
    ])

    expect(exitCode).toBe(1)
    expect(stderr).toContain('duplicate operationId ReadThing: GET /default and POST /named/{id}')
  })
})
