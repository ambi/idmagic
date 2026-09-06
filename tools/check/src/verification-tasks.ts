/**
 * Read the shape of the verification suite out of the command map.
 *
 * An aggregate gate names its members in one of two ways. `depends` builds one
 * dependency graph and stops at the first failure; a `run` that invokes
 * `mise run -c <task> ::: <task>` runs the same members and reports every one
 * that failed. Both forms are in use, and three consumers need the membership
 * regardless of which form a task chose: the timing table walks it, the
 * primary-use-case checker asks whether a declared test task is reachable from
 * the standard suite, and the command-map tests assert that the parallel and
 * serial suites still carry the same set. Reading only `depends` would report
 * an aggregate written the second way as having no members at all, which is
 * the failure mode that says nothing and passes everything.
 */

export type MiseTask = {
  depends?: unknown
  run?: unknown
}

export type MiseTasks = Record<string, MiseTask>

export function parseMiseTasks(miseToml: string): MiseTasks {
  const document = Bun.TOML.parse(miseToml) as { tasks?: MiseTasks }
  return document.tasks ?? {}
}

/**
 * The task names one `mise run` invocation names. Flags are skipped, `:::`
 * starts the next task, and everything after `--` is an argument the task
 * receives rather than another task.
 */
function invokedTasks(command: string): string[] {
  // A continued line is one command; joining first keeps the trailing `\` from
  // being read as a token where a task name is expected.
  const normalized = command.replace(/\\\r?\n/g, ' ')
  const names: string[] = []
  for (const segment of normalized.split(/\bmise\s+run\s+/).slice(1)) {
    // One invocation ends at the shell operator that starts the next command.
    const invocation = (segment.split(/[\n;&|]/)[0] ?? '').trim()
    let expectTask = true
    for (const token of invocation.split(/\s+/)) {
      if (token === '') continue
      if (token === ':::') {
        expectTask = true
        continue
      }
      if (token === '--') break
      if (token.startsWith('-')) continue
      if (expectTask && /^[a-z][a-z0-9-]*$/.test(token)) names.push(token)
      expectTask = false
    }
  }
  return names
}

function commands(run: unknown): string[] {
  if (typeof run === 'string') return [run]
  if (Array.isArray(run)) return run.filter((entry): entry is string => typeof entry === 'string')
  return []
}

/** The tasks one task runs directly, in declaration order and without repeats. */
export function directTasks(tasks: MiseTasks, name: string): string[] {
  const task = tasks[name]
  if (!task) return []
  const names: string[] = []
  const add = (candidate: string) => {
    if (!names.includes(candidate)) names.push(candidate)
  }
  if (Array.isArray(task.depends)) {
    for (const entry of task.depends) if (typeof entry === 'string') add(entry)
  }
  for (const command of commands(task.run)) {
    for (const invoked of invokedTasks(command)) add(invoked)
  }
  return names
}

/** Every task reachable from an entry point, the entry point included. */
export function taskClosure(tasks: MiseTasks, entry: string): Set<string> {
  const reached = new Set<string>()
  const visit = (name: string): void => {
    if (reached.has(name)) return
    reached.add(name)
    for (const next of directTasks(tasks, name)) visit(next)
  }
  visit(entry)
  return reached
}
