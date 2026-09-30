#!/usr/bin/env bun

import { resolve } from 'node:path'
import { diffWorkspaceSpecifications, formatSpecificationDiff } from '../../check/src/spec-diff.ts'

const root = resolve(import.meta.dir, '../../..')
const ref = process.argv[2] ?? 'main'
console.log(formatSpecificationDiff(await diffWorkspaceSpecifications(root, ref), ref))
