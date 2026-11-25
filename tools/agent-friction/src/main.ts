#!/usr/bin/env bun

/**
 * `git log` の `Agent-Friction:` トレーラーを対象ごとに集計して表示する。
 *
 * `Agent-Friction-Reviewed:` トレーラーで判断した対象は、そのコミットより前の記録を数えない。
 * 引数に Git のリビジョン範囲を渡すと、その範囲のコミットだけを読む。省けば HEAD までの全履歴を読む。
 * 報告であってゲートではないので、所見があっても終了コードは 0 にする。
 */

import { existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { parseMiseTasks } from '../../check/src/verification-tasks.ts'
import {
  type FrictionTarget,
  formatFrictionReport,
  type LogRecord,
  summarizeFriction,
  unreviewedEntries,
} from './friction.ts'

const root = resolve(import.meta.dir, '../../..')
const range = process.argv.slice(2)

const RECORD = '\x1e'
const FIELD = '\x1f'
const VALUE = '\x1d'

const log = Bun.spawnSync(
  [
    'git',
    'log',
    `--format=${RECORD}%H${FIELD}%cs${FIELD}%(trailers:key=Agent-Friction,valueonly,unfold,separator=%x1d)${FIELD}%(trailers:key=Agent-Friction-Reviewed,valueonly,separator=%x1d)`,
    ...range,
  ],
  { cwd: root, stdout: 'pipe', stderr: 'pipe' },
)
if (log.exitCode !== 0) {
  process.stderr.write(log.stderr)
  process.exit(2)
}

const records: LogRecord[] = log.stdout
  .toString()
  .split(RECORD)
  .filter((record) => record.trim() !== '')
  .map((record) => {
    const [sha = '', date = '', frictions = '', reviewed = ''] = record.split(FIELD)
    return {
      sha,
      date,
      frictions: frictions
        .split(VALUE)
        .map((value) => value.trim())
        .filter((value) => value !== ''),
      reviewed: reviewed
        .split(VALUE)
        .map((value) => value.trim())
        .filter((value) => value !== ''),
    }
  })

// git log は新しい順に返す。判断済みの対象を外したうえで、読む人が起きた順に追えるよう古い順に返る。
const entries = unreviewedEntries(records)

const tasks = parseMiseTasks(await Bun.file(resolve(root, 'mise.toml')).text())
const exists = (target: FrictionTarget) =>
  target.kind === 'task' ? target.task in tasks : existsSync(resolve(root, target.path))

process.stdout.write(formatFrictionReport(summarizeFriction(entries, exists)))
