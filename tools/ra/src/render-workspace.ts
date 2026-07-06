#!/usr/bin/env bun
import { loadWorkspaceConfig, rootPath, runTool } from './workspace.ts'

const config = await loadWorkspaceConfig()

for (const app of config.apps) {
  const artifacts = app.artifacts ?? {}
  if (artifacts.html) {
    await runTool([
      'scl-to-html/src/main.ts',
      '--scl',
      rootPath(app.scl),
      '--title',
      app.name,
      '--out',
      rootPath(artifacts.html),
    ])
  }
  if (artifacts.fullHtml) {
    const args = [
      'scl-to-html/src/main.ts',
      '--scl',
      rootPath(app.scl),
      '--title',
      app.name,
      '--out',
      rootPath(artifacts.fullHtml),
    ]
    if (app.decisions) args.push('--decisions', rootPath(app.decisions))
    if (app.workItems) args.push('--work-items', rootPath(app.workItems))
    if (app.architecture) args.push('--architecture', rootPath(app.architecture))
    await runTool(args)
  }
  if (artifacts.jsonSchema) {
    await runTool([
      'scl-to-jsonschema/src/main.ts',
      '--scl',
      rootPath(app.scl),
      '--out',
      rootPath(artifacts.jsonSchema),
    ])
  }
  if (artifacts.openApi) {
    await runTool([
      'scl-to-openapi/src/main.ts',
      '--scl',
      rootPath(app.scl),
      '--out',
      rootPath(artifacts.openApi),
    ])
  }
}

for (const spec of config.toolSpecs ?? []) {
  const title = spec.split('/').at(-3) ?? spec
  const out = spec.replace(/\/scl\.yaml$/, `/${title}.html`)
  await runTool([
    'scl-to-html/src/main.ts',
    '--scl',
    rootPath(spec),
    '--title',
    title,
    '--out',
    rootPath(out),
  ])
}
