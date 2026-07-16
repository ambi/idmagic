import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import {
  collectArchitectureWorkspace,
  evaluateArchitectureWorkspace,
  isProductionSource,
} from './architecture-workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

describe('architecture production import graph', () => {
  const architecture = {
    modules: {
      usecases: { path: 'backend/accounts/usecases', depends_on: ['domain'] },
      domain: { path: 'backend/accounts/domain', depends_on: [] },
      adapters: { path: 'backend/accounts/adapters', depends_on: [] },
      ui: { path: 'frontend/src', depends_on: [{ module: 'ui-shared' }] },
      'ui-shared': { path: 'frontend/shared', depends_on: [] },
      'ui-types': { path: 'frontend/types.ts', depends_on: [] },
    },
  }

  it('accepts declared Go and aliased TypeScript module imports', () => {
    const findings = evaluateArchitectureWorkspace(architecture, {
      goModulePath: 'example.test/app',
      tsAliases: { '@shared/*': ['frontend/shared/*'] },
      files: [
        {
          path: 'backend/accounts/usecases/create.go',
          content: 'package usecases\nimport "example.test/app/backend/accounts/domain"\n',
        },
        {
          path: 'frontend/src/page.tsx',
          content: "import { Button } from '@shared/button'\n",
        },
      ],
    })

    expect(findings).toEqual([])
  })

  it('reports a production import missing from depends_on', () => {
    const findings = evaluateArchitectureWorkspace(architecture, {
      goModulePath: 'example.test/app',
      files: [
        {
          path: 'backend/accounts/usecases/create.go',
          content: 'package usecases\nimport "example.test/app/backend/accounts/adapters/http"\n',
        },
      ],
    })

    expect(findings).toEqual([
      expect.objectContaining({
        kind: 'import',
        path: 'backend/accounts/usecases/create.go',
        message: expect.stringContaining("imports 'adapters'"),
      }),
    ])
  })

  it('reports production sources and workspace-local import targets outside declared modules', () => {
    const findings = evaluateArchitectureWorkspace(architecture, {
      goModulePath: 'example.test/app',
      files: [
        {
          path: 'backend/unmapped/job.go',
          content: 'package unmapped\n',
        },
        {
          path: 'backend/accounts/usecases/create.go',
          content: 'package usecases\nimport "example.test/app/backend/unknown"\n',
        },
      ],
    })

    expect(findings).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          kind: 'import',
          path: 'backend/unmapped/job.go',
          message: expect.stringContaining('belongs to no declared module'),
        }),
        expect.objectContaining({
          kind: 'import',
          path: 'backend/accounts/usecases/create.go',
          message: expect.stringContaining("import target 'backend/unknown'"),
        }),
      ]),
    )
  })

  it('resolves extensionless TypeScript imports to file modules', () => {
    const findings = evaluateArchitectureWorkspace(architecture, {
      files: [
        {
          path: 'frontend/src/page.tsx',
          content: "import type { User } from '../types'\n",
        },
      ],
    })

    expect(findings).toEqual([
      expect.objectContaining({ message: expect.stringContaining("imports 'ui-types'") }),
    ])
  })

  it('excludes generated, vendored and test sources', () => {
    for (const path of [
      'vendor/pkg/file.go',
      'node_modules/pkg/index.ts',
      'backend/generated/client.go',
      'backend/sqlc/queries.go',
      'backend/persistence/pgfixtures/data.go',
      'frontend/src/routeTree.gen.ts',
      'backend/domain/model_test.go',
      'frontend/src/page.test.tsx',
      'frontend/src/page.spec.ts',
    ]) {
      expect(isProductionSource(path)).toBe(false)
    }
  })
})

describe('architecture complexity budgets', () => {
  const lines = (count: number) =>
    `${Array.from({ length: count }, (_, i) => `line ${i}`).join('\n')}\n`
  const architecture = {
    modules: { ui: { path: 'frontend/src', depends_on: [] } },
    complexity: {
      budgets: {
        pages: {
          metric: 'source_lines',
          include: ['frontend/src/pages/**/*.tsx'],
          exclude: ['**/*.generated.tsx'],
          limit: 4,
        },
        state: {
          metric: 'react_local_state_hooks',
          include: ['frontend/src/pages/**/*.tsx'],
          limit: 1,
        },
      },
      debts: {
        legacy: {
          budget: 'pages',
          path: 'frontend/src/pages/legacy.tsx',
          ceiling: 6,
          expires_at: '2026-10-01',
        },
      },
    },
  }

  it('accepts bounded, unexpired ratchet debt', () => {
    const findings = evaluateArchitectureWorkspace(
      architecture,
      { files: [{ path: 'frontend/src/pages/legacy.tsx', content: lines(6) }] },
      { today: '2026-07-17' },
    )
    expect(findings).toEqual([])
  })

  it('rejects an undeclared overage, a ceiling increase, and expired debt', () => {
    const findings = evaluateArchitectureWorkspace(
      architecture,
      {
        files: [
          { path: 'frontend/src/pages/new.tsx', content: lines(5) },
          { path: 'frontend/src/pages/legacy.tsx', content: lines(7) },
        ],
      },
      { today: '2026-10-02' },
    )

    expect(findings.map((finding) => finding.message)).toEqual([
      expect.stringContaining('above debt ceiling 6'),
      expect.stringContaining('expired on 2026-10-01'),
      expect.stringContaining('without declared debt'),
    ])
  })

  it('counts React local state hooks and honors excluded globs', () => {
    const findings = evaluateArchitectureWorkspace(architecture, {
      files: [
        {
          path: 'frontend/src/pages/stateful.tsx',
          content: 'const [a] = useState(1)\nconst [b] = useState<string>("x")\n',
        },
        { path: 'frontend/src/pages/large.generated.tsx', content: lines(20) },
        { path: 'frontend/src/pages/legacy.tsx', content: lines(4) },
      ],
    })
    expect(findings).toEqual([
      expect.objectContaining({ kind: 'complexity', path: 'frontend/src/pages/stateful.tsx' }),
    ])
  })
})
describe('collectArchitectureWorkspace', () => {
  it('normalizes the Go module and reads only production source files', async () => {
    const root = await mkdtemp(join(tmpdir(), 'ra-architecture-workspace-'))
    cleanup.push(root)
    await mkdir(join(root, 'backend', 'domain'), { recursive: true })
    await mkdir(join(root, 'node_modules', 'ignored'), { recursive: true })
    await writeFile(join(root, 'go.mod'), 'module example.test/demo\n\ngo 1.24\n')
    await writeFile(join(root, 'backend', 'domain', 'model.go'), 'package domain\n')
    await writeFile(join(root, 'backend', 'domain', 'model_test.go'), 'package domain\n')
    await writeFile(join(root, 'node_modules', 'ignored', 'index.ts'), 'export {}\n')

    const snapshot = await collectArchitectureWorkspace(root)

    expect(snapshot.goModulePath).toBe('example.test/demo')
    expect(snapshot.files.map((file) => file.path)).toEqual(['backend/domain/model.go'])
  })

  it('collects tsconfig path aliases relative to each config directory', async () => {
    const root = await mkdtemp(join(tmpdir(), 'ra-architecture-workspace-'))
    cleanup.push(root)
    await mkdir(join(root, 'frontend', 'src'), { recursive: true })
    await writeFile(
      join(root, 'frontend', 'tsconfig.json'),
      `{
        // JSONC is the normal tsconfig syntax.
        "compilerOptions": {
          "baseUrl": ".",
          "paths": { "@/*": ["src/*"], },
        },
      }`,
    )
    await writeFile(join(root, 'frontend', 'src', 'page.ts'), 'export {}\n')

    const snapshot = await collectArchitectureWorkspace(root)

    expect(snapshot.tsAliases).toEqual({ '@/*': ['frontend/src/*'] })
  })
})
