import {
  CONTEXT_DOCUMENTS,
  SYSTEM_DOCUMENT_DIRECTORIES,
} from '../../workspace/src/document-layout.ts'

export type DocumentLayoutFinding = {
  path: string
  message: string
}

/** 定義済みの文書が、人向けの配置図にも現れることを確かめる。 */
export function verifyDocumentLayout(source: string): DocumentLayoutFinding[] {
  const documented = documentedPaths(source)
  return requiredDocumentPaths()
    .filter((path) => !documented.has(path))
    .map((path) => ({ path, message: `配置図に定義済み文書のパスがない: ${path}` }))
}

function requiredDocumentPaths(): string[] {
  const systemDocuments = SYSTEM_DOCUMENT_DIRECTORIES.flatMap(({ directory, names }) =>
    names.map((name) => `${directory}/${name}`),
  )
  const contextDocuments = CONTEXT_DOCUMENTS.map((name) => `docs/domain/<context>/${name}`)
  return [...systemDocuments, ...contextDocuments]
}

function documentedPaths(source: string): Set<string> {
  const lines = layoutLines(source)
  const paths = new Set<string>()
  const directories: Array<{ indentation: number; path: string }> = []

  for (const line of lines) {
    if (line.trim() === '') continue
    const indentation = line.length - line.trimStart().length
    const entry = line.trim().replace(/\s+#.*$/, '')

    while (
      directories.length > 0 &&
      directories[directories.length - 1]!.indentation >= indentation
    ) {
      directories.pop()
    }

    if (indentation === 0) {
      directories.length = 0
      if (entry !== 'docs/') continue
      directories.push({ indentation, path: 'docs' })
      continue
    }

    const parent = directories[directories.length - 1]
    if (parent === undefined || !parent.path.startsWith('docs')) continue

    const isDirectory = entry.endsWith('/')
    const path = `${parent.path}/${isDirectory ? entry.slice(0, -1) : entry}`
    paths.add(path)
    if (isDirectory) directories.push({ indentation, path })
  }

  return paths
}

function layoutLines(source: string): string[] {
  const placement = source.indexOf('## 1. 配置')
  if (placement < 0) return []
  const afterPlacement = source.slice(placement)
  const opening = afterPlacement.indexOf('```text')
  if (opening < 0) return []
  const body = afterPlacement.slice(opening + '```text'.length)
  const closing = body.indexOf('```')
  if (closing < 0) return []
  return body.slice(0, closing).split('\n')
}
