import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import {
  buildSclWorkspaceIndex,
  type SclReferenceError,
  type SclWorkspaceIndex,
} from '../../yaml-check/src/scl-element-reference.ts'
import { rootPath, type WorkspaceConfig } from './workspace.ts'

export async function loadWorkspaceSclIndex(
  config: WorkspaceConfig,
): Promise<{ index: SclWorkspaceIndex; errors: SclReferenceError[] }> {
  const contexts = new Map<string, Record<string, unknown>>()
  const errors: SclReferenceError[] = []

  const merge = (built: ReturnType<typeof buildSclWorkspaceIndex>) => {
    if (!built.ok) {
      errors.push(...built.errors)
      return
    }
    for (const [context, document] of built.index.contexts) {
      if (contexts.has(context)) {
        errors.push({ code: 'context_mismatch', message: `duplicate SCL context '${context}'` })
      } else {
        contexts.set(context, document)
      }
    }
  }

  for (const app of config.apps) {
    const rootFile = rootPath(app.scl)
    const root = Bun.YAML.parse(await readFile(rootFile, 'utf8')) as Record<string, unknown>
    const documents: Record<string, unknown> = {}
    for (const [context, entryValue] of Object.entries(
      (root.context_map as Record<string, unknown> | undefined) ?? {},
    )) {
      const path = (entryValue as { path?: unknown } | null)?.path
      if (typeof path !== 'string') continue
      try {
        documents[context] = Bun.YAML.parse(
          await readFile(resolve(dirname(rootFile), path), 'utf8'),
        )
      } catch {
        // The shared index builder emits a context-qualified unavailable error.
      }
    }
    merge(buildSclWorkspaceIndex(root, documents))
  }

  for (const spec of config.toolSpecs ?? []) {
    const document = Bun.YAML.parse(await readFile(rootPath(spec), 'utf8'))
    merge(buildSclWorkspaceIndex(document, {}))
  }

  return { index: { root: {}, contexts }, errors }
}
