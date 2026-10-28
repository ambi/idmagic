import type { GoPackage } from './changed-packages.ts'
import type { Plan } from './plan.ts'

export type Gate = { layer: keyof Plan; command: string[] }

/**
 * The commands a plan runs. Go is narrowed to the selected packages; the other
 * layers are cheap enough to run whole once selected. A Go change that may
 * reach every package runs the whole-module lint and tests.
 *
 * Go tests use the ordinary build. The race detector runs on a schedule
 * against main, and on request while building concurrent code.
 */
export function gatesFor(plan: Plan, packages: readonly GoPackage[]): Gate[] {
  const gates: Gate[] = []
  if (plan.check.selected) gates.push({ layer: 'check', command: ['mise', 'run', 'check'] })
  if (plan.go.selected && plan.go.all) {
    gates.push({ layer: 'go', command: ['mise', 'run', 'lint-go'] })
    gates.push({ layer: 'go', command: ['mise', 'run', 'test-go'] })
  } else if (plan.go.selected) {
    const directories = new Map(packages.map((one) => [one.importPath, one.dir]))
    gates.push({
      layer: 'go',
      command: [
        'golangci-lint',
        'run',
        ...plan.go.packages.map((name) => `./${directories.get(name) ?? name}`),
      ],
    })
    gates.push({
      layer: 'go',
      command: [
        'bun',
        'run',
        'tools/verification/src/main.ts',
        '--report',
        '.verification/test-go-selected.json',
        '--',
        ...plan.go.packages,
      ],
    })
  }
  if (plan.ui.selected) gates.push({ layer: 'ui', command: ['mise', 'run', 'verify-ui'] })
  if (plan.tools.selected) {
    for (const task of ['test-tools', 'typecheck-tools', 'lint-tools'])
      gates.push({ layer: 'tools', command: ['mise', 'run', task] })
  }
  return gates
}
