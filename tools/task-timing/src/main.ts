#!/usr/bin/env bun

/**
 * Time every gate of a verification suite, one at a time.
 *
 * The suite normally runs in parallel, which is the right way to run it and
 * the wrong way to measure it: the wall time of a parallel run says what the
 * machine had spare, not what each gate costs. Running the members serially
 * costs more once and produces a number that can be compared with the same
 * number taken after a change. Output of a passing gate is dropped, because
 * the table is the result; a failing gate keeps its output, because a suite
 * that failed has something to read.
 */

import { resolve } from 'node:path'
import { directTasks, parseMiseTasks } from '../../check/src/verification-tasks.ts'
import { formatTimingTable, type TimingRow } from './timing.ts'

const root = resolve(import.meta.dir, '../../..')
const entry = process.argv[2] ?? 'verify'

const tasks = parseMiseTasks(await Bun.file(resolve(root, 'mise.toml')).text())
if (!tasks[entry]) {
  console.error(`time-verify: mise.toml declares no task named ${entry}`)
  process.exit(2)
}

const members = directTasks(tasks, entry)
if (members.length === 0) {
  console.error(`time-verify: ${entry} has no member tasks to time`)
  process.exit(2)
}

const rows: TimingRow[] = []
for (const task of members) {
  process.stderr.write(`== ${task}\n`)
  const started = Bun.nanoseconds()
  const process_ = Bun.spawn(['mise', 'run', task], {
    cwd: root,
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(process_.stdout).text(),
    new Response(process_.stderr).text(),
    process_.exited,
  ])
  const milliseconds = (Bun.nanoseconds() - started) / 1_000_000
  rows.push({ task, milliseconds, ok: code === 0 })
  if (code !== 0) process.stderr.write(`${stdout}${stderr}`)
}

process.stdout.write(formatTimingTable(rows))
process.exit(rows.every((row) => row.ok) ? 0 : 1)
