const ciJobs = {
  go: ['go-lint', 'go-test'],
  ui: ['ui'],
  browser: ['ui-e2e'],
  schema: ['schema-convergence'],
} as const

type Group = keyof typeof ciJobs
type Decision = { selected: boolean; reasons: string[] }
export type CIPlan = { groups: Record<Group, Decision> }

export function planCI(paths: string[] | null): CIPlan {
  const groups: CIPlan['groups'] = {
    go: { selected: false, reasons: [] },
    ui: { selected: false, reasons: [] },
    browser: { selected: false, reasons: [] },
    schema: { selected: false, reasons: [] },
  }
  const all = Object.keys(groups) as Group[]
  const inputs = paths?.length ? [...new Set(paths)].sort() : [null]
  const excluded: string[] = []
  for (const path of inputs) {
    let selected: readonly Group[] = all
    if (
      path?.endsWith('.md') &&
      (path.startsWith('work-items/') || path.startsWith('docs/development/'))
    ) {
      excluded.push(path)
      continue
    }
    if (path?.startsWith('frontend/')) selected = ['ui', 'browser']
    else if (
      path?.startsWith('backend/') &&
      !path.startsWith('backend/shared/') &&
      !path.startsWith('backend/cmd/')
    ) {
      selected = ['go', 'browser']
    }
    const reason =
      path ??
      (paths === null ? 'comparison unavailable; run all groups' : 'empty diff; run all groups')
    for (const group of selected) {
      groups[group].selected = true
      groups[group].reasons.push(reason)
    }
  }
  for (const decision of Object.values(groups)) {
    if (!decision.selected) {
      decision.reasons = excluded.length
        ? [`documentation only or unrelated group: ${excluded.join(', ')}`]
        : ['no changed input affects this group']
    }
  }
  return { groups }
}

/** rename を delete/add として読み、旧パスの影響を消さない。 */
export function planCIRevision(root: string, base?: string, head?: string): CIPlan {
  try {
    if (!base || !head) return planCI(null)
    const git = (...args: string[]) => {
      const result = Bun.spawnSync(['git', ...args], { cwd: root, stderr: 'pipe' })
      if (result.exitCode !== 0) throw new Error('git comparison failed')
      return result.stdout.toString()
    }
    const before = git('rev-parse', '--verify', '--end-of-options', `${base}^{commit}`).trim()
    const after = git('rev-parse', '--verify', '--end-of-options', `${head}^{commit}`).trim()
    return planCI(
      git('diff', '--name-only', '--no-renames', '-z', before, after, '--')
        .split('\0')
        .filter(Boolean),
    )
  } catch {
    return planCI(null)
  }
}

function object(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function validPlan(value: unknown): value is CIPlan {
  if (!object(value) || !object(value.groups)) return false
  const groups = value.groups
  return Object.keys(ciJobs).every((group) => {
    const decision = groups[group]
    return (
      object(decision) &&
      typeof decision.selected === 'boolean' &&
      Array.isArray(decision.reasons) &&
      decision.reasons.length > 0 &&
      decision.reasons.every((reason) => typeof reason === 'string' && reason.trim().length > 0)
    )
  })
}

export function evaluateCIResults(needs: unknown): { success: boolean; lines: string[] } {
  if (
    !object(needs) ||
    !object(needs.plan) ||
    needs.plan.result !== 'success' ||
    !object(needs.plan.outputs) ||
    typeof needs.plan.outputs.plan !== 'string'
  ) {
    return { success: false, lines: ['CI plan failed or is missing'] }
  }
  let plan: unknown
  try {
    plan = JSON.parse(needs.plan.outputs.plan)
  } catch {
    return { success: false, lines: ['CI plan is invalid JSON'] }
  }
  if (!validPlan(plan)) return { success: false, lines: ['CI plan is invalid'] }
  const lines: string[] = []
  let success = true
  const requireResult = (job: string, selected: boolean, reasons: string[]) => {
    const result = object(needs[job]) ? needs[job].result : undefined
    const expected = selected ? 'success' : 'skipped'
    if (result !== expected) success = false
    lines.push(
      `${job}: ${result ?? 'missing'}; ${selected ? 'selected' : 'not selected'}; ${reasons.join('; ')}`,
    )
  }
  requireResult('spec', true, ['repository verification always runs'])
  for (const group of Object.keys(ciJobs) as Group[]) {
    const decision = plan.groups[group]
    for (const job of ciJobs[group]) requireResult(job, decision.selected, decision.reasons)
  }
  return { success, lines }
}
