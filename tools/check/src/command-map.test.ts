import { describe, expect, it } from 'bun:test'
import { invokedRecipes, recipeNames, verifyCommandMap } from './command-map.ts'

const justfile = [
  'golangci_cache := env("CACHE", "/tmp/cache")',
  '',
  '# Run everything.',
  'verify: check test-go',
  '',
  '# Check one package.',
  'test-go-package package:',
  '    go test {{package}}',
  '',
  '# Check the workspace.',
  'check:',
  '    bun run check.ts',
  '',
  'test-go:',
  '    go test ./...',
].join('\n')

describe('recipeNames', () => {
  it('reads plain, dependent, and parameterized recipes but not assignments', () => {
    expect([...recipeNames(justfile)].sort()).toEqual([
      'check',
      'test-go',
      'test-go-package',
      'verify',
    ])
  })
})

describe('invokedRecipes', () => {
  it('finds recipes called from a workflow step', () => {
    expect(invokedRecipes('run: |\n  just check\n  just test-go-package ./backend\n')).toEqual([
      'check',
      'test-go-package',
    ])
  })
})

describe('verifyCommandMap', () => {
  it('accepts a workflow that only calls declared recipes', () => {
    const workflow = { file: 'ci.yaml', source: 'run: just check\nrun: just verify\n' }
    expect(verifyCommandMap(justfile, [workflow])).toEqual([])
  })

  it('reports a recipe the justfile no longer declares', () => {
    const workflow = {
      file: 'ci.yaml',
      source: 'run: |\n  just check\n  just traceability-strict\n',
    }
    expect(verifyCommandMap(justfile, [workflow])).toEqual([
      { file: 'ci.yaml', recipe: 'traceability-strict' },
    ])
  })

  it('reports each missing recipe once', () => {
    const workflows = [
      { file: 'a.yaml', source: 'run: just gone\n' },
      { file: 'b.yaml', source: 'run: just gone\n' },
    ]
    expect(verifyCommandMap(justfile, workflows)).toEqual([{ file: 'a.yaml', recipe: 'gone' }])
  })
})
