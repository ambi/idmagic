export interface DirectoryListing {
  directory: string
  files: string[]
}

export const CONTEXT_DOCUMENTS = [
  'README.md',
  'glossary.md',
  'standards.md',
  'states.md',
  'decisions.md',
  'internals.md',
  'scenarios.feature.md',
] as const

export const ROOT_DOCUMENTS = [
  'README.md',
  'product-overview.md',
  'glossary.md',
  'standards.md',
  'structure.md',
  'scenarios.feature.md',
] as const

/** 正準文書を上位から読む順序で定義する。 */
export const SYSTEM_DOCUMENT_DIRECTORIES = [
  { directory: 'docs', names: ROOT_DOCUMENTS },
  {
    directory: 'docs/requirements',
    names: ['README.md', 'functional.md', 'quality.md', 'constraints.md'],
  },
  {
    directory: 'docs/architecture',
    names: [
      'README.md',
      'system-context.md',
      'logical.md',
      'runtime.md',
      'deployment.md',
      'decisions.md',
    ],
  },
  { directory: 'docs/design', names: ['README.md'] },
  {
    directory: 'docs/design/application',
    names: ['README.md', 'api-rules.md', 'design-rules.md', 'user-interface.md'],
  },
  { directory: 'docs/design/data', names: ['README.md', 'database.md', 'lifecycle.md'] },
  {
    directory: 'docs/design/infrastructure',
    names: ['README.md', 'platform.md', 'network.md'],
  },
  {
    directory: 'docs/design/security',
    names: ['README.md', 'threat-model.md', 'authorization.md', 'secrets.md'],
  },
  {
    directory: 'docs/design/reliability',
    names: ['README.md', 'availability.md', 'recovery.md'],
  },
  {
    directory: 'docs/design/performance',
    names: ['README.md', 'capacity.md', 'scaling.md'],
  },
  {
    directory: 'docs/design/observability',
    names: ['README.md', 'monitoring.md', 'logging.md', 'tracing.md'],
  },
  { directory: 'docs/verification', names: ['README.md', 'system-acceptance.md'] },
  {
    directory: 'docs/operations',
    names: ['README.md', 'service-management.md', 'maintenance.md'],
  },
] as const

const SYSTEM_DOCUMENTS_BY_DIRECTORY = new Map<string, readonly string[]>(
  SYSTEM_DOCUMENT_DIRECTORIES.map(({ directory, names }) => [directory, names]),
)

export const SYSTEM_DOCUMENT_PATHS = SYSTEM_DOCUMENT_DIRECTORIES.flatMap(({ directory, names }) =>
  names.map((name) => `${directory}/${name}`),
)

/** 内容に応じた任意名を許し、閉じたファイル集合の対象にしない段。 */
export const FREELY_NAMED_DOCUMENT_DIRECTORIES = new Set([
  'docs/contexts',
  'docs/development',
  'docs/runbooks',
  'docs/releases',
])

export function canonicalDocumentNames(directory: string): readonly string[] | undefined {
  return SYSTEM_DOCUMENTS_BY_DIRECTORY.get(directory)
}
