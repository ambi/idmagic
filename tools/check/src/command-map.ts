/**
 * Keep the command map and the pipelines that call it in agreement.
 *
 * A task removed during an ordinary refactor takes every caller with it, and a
 * caller living in CI keeps passing locally: `mise run verify` is green while
 * the pipeline stops because the task is missing. Nothing else
 * in the repository can see that gap, so it is checked here.
 *
 * Only the direction that can silently break a pipeline is checked. A task no
 * CI job calls is not a defect — the command map serves people too.
 */

export type CommandMapFinding = {
  file: string
  task: string
}

/** Task names declared in mise.toml. */
export function taskNames(miseToml: string): Set<string> {
  const document = Bun.TOML.parse(miseToml) as { tasks?: Record<string, unknown> }
  return new Set(Object.keys(document.tasks ?? {}))
}

/** Tasks a workflow invokes as `mise run <task>`. */
export function invokedTasks(workflow: string): string[] {
  return [...workflow.matchAll(/(?:^|\s)mise\s+run\s+([a-z][a-z0-9-]*)/g)].map(
    (match) => match[1] ?? '',
  )
}

export function verifyCommandMap(
  miseToml: string,
  workflows: Array<{ file: string; source: string }>,
): CommandMapFinding[] {
  const declared = taskNames(miseToml)
  const findings: CommandMapFinding[] = []
  for (const workflow of workflows) {
    for (const task of invokedTasks(workflow.source)) {
      if (!declared.has(task) && !findings.some((one) => one.task === task)) {
        findings.push({ file: workflow.file, task })
      }
    }
  }
  return findings
}
