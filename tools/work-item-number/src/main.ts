#!/usr/bin/env bun

/**
 * 未使用の識別番号を一つ出力する。
 *
 * 走査と乱数はここだけが持つ。`work-items/` と `work-items/done/` の両方を読むのは、
 * 完了した記録の番号も参照として残っており、再利用できないためである。
 */

import { readdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pickWorkItemNumber, usedWorkItemNumbers } from './work-item-number.ts'

const root = resolve(import.meta.dir, '../../..')

const fileNames: string[] = []
for (const directory of ['work-items', 'work-items/done']) {
  fileNames.push(...(await readdir(resolve(root, directory))))
}

process.stdout.write(`${pickWorkItemNumber(usedWorkItemNumbers(fileNames), Math.random)}\n`)
