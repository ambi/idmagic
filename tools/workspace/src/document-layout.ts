export interface DirectoryListing {
  directory: string
  files: string[]
}

export const ROOT_DOCUMENTS = ['README.md'] as const

/** ドメイン全体を対象とし、モジュールのディレクトリより上に置く文書。 */
export const DOMAIN_DOCUMENTS = [
  'README.md',
  'glossary.md',
  'standards.md',
  'structure.md',
  'scenarios.feature.md',
] as const

/** 一次情報文書を上位から読む順序で定義する。 */
export const SYSTEM_DOCUMENT_DIRECTORIES = [
  { directory: 'docs', names: ROOT_DOCUMENTS },
  {
    directory: 'docs/formats',
    names: [
      'README.md',
      'documentation-guide.md',
      'specification-format.md',
      'work-item-format.md',
    ],
  },
  { directory: 'docs/domain', names: DOMAIN_DOCUMENTS },
  {
    directory: 'docs/requirements',
    names: ['README.md', 'product-overview.md', 'functional.md', 'quality.md'],
  },
  {
    directory: 'docs/design/architecture',
    names: [
      'README.md',
      'system-boundary.md',
      'constraints.md',
      'strategy.md',
      'logical.md',
      'runtime.md',
      'deployment.md',
      'decisions.md',
      'risks.md',
    ],
  },
  { directory: 'docs/design', names: ['README.md'] },
  {
    directory: 'docs/design/application',
    names: [
      'README.md',
      'api-guidelines.md',
      'design-guidelines.md',
      'frontend.md',
      'user-interface.md',
    ],
  },
  {
    directory: 'docs/design/data',
    names: ['README.md', 'database.md', 'schema-management.md', 'lifecycle.md'],
  },
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
  {
    directory: 'docs/design/verification',
    names: ['README.md', 'system-acceptance.md', 'security.md'],
  },
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

/**
 * 内容に応じた任意名を許し、閉じたファイル集合の対象にしない段。`docs/domain` は
 * 直下のファイル集合が閉じており、配下のディレクトリ名だけが自由なので、ここには載せない。
 * 配下のモジュールの段は `documentAllowance` が判定する。
 */
export const FREELY_NAMED_DOCUMENT_DIRECTORIES = new Set([
  'docs/development',
  'docs/runbooks',
  'docs/releases',
])

export function canonicalDocumentNames(directory: string): readonly string[] | undefined {
  return SYSTEM_DOCUMENTS_BY_DIRECTORY.get(directory)
}

/**
 * モジュールの直下に置ける文書。判断と仕組みは `design/` へ、規則は機能スライスへ置く。
 * `quality.md` は、システムの品質要求のうちこのモジュールに割り当てた分を書く仕様である。
 */
export const CONTEXT_DOCUMENTS = ['README.md', 'glossary.md', 'standards.md', 'quality.md'] as const

/** モジュールの内部設計の段に置く固定の文書。横断的概念は任意の名前で並べる。 */
export const DESIGN_DOCUMENTS = ['README.md', 'decisions.md'] as const

/** 機能スライスの固定の文書。長くなった機能仕様の章は任意の名前で並べる。 */
export const FEATURE_SLICE_DOCUMENTS = ['README.md', 'design.md', 'acceptance.feature.md'] as const

/**
 * 任意の名前を許す段でも使えない名前。ファイル種別ごとの文書を章として置くと、
 * 一つの機能を種別ごとのファイルに散らす構造へ戻ってしまう。用語と標準はモジュールの直下に置く。
 */
const RESERVED_FREE_NAMES = new Set([
  'states.md',
  'decisions.md',
  'internals.md',
  'scenarios.feature.md',
  'glossary.md',
  'standards.md',
])

/** 任意の名前の文書は、ケバブケースの Markdown に限る。`Readme.md` のような打ち間違いを通さない。 */
const FREE_DOCUMENT_NAME = /^[a-z0-9]+(?:-[a-z0-9]+)*\.md$/

/** ある段に置ける文書。`freeNames` が真なら、固定の名前に加えて任意の名前の章も置ける。 */
export interface DocumentAllowance {
  names: readonly string[]
  freeNames: boolean
}

/** 段の集合から読み取った、名前の判定に要る事実。 */
export interface DocumentSetView {
  /**
   * `design/README.md` を持つモジュールの名前。この印のないモジュールの段には、どの文書も置けない。
   * 設定を読まずに同じ判定ができるよう、ファイルの有無で決める。
   */
  featureContexts: ReadonlySet<string>
  /** 子のディレクトリを持つ段。これが機能群と機能スライスを分ける。 */
  parents: ReadonlySet<string>
}

export function describeDocumentSet(listings: readonly DirectoryListing[]): DocumentSetView {
  const featureContexts = new Set<string>()
  const parents = new Set<string>()
  for (const listing of listings) {
    const design = listing.directory.match(/^docs\/domain\/([^/]+)\/design$/)?.[1]
    if (design && listing.files.includes('README.md')) featureContexts.add(design)
    const parent = listing.directory.slice(0, Math.max(0, listing.directory.lastIndexOf('/')))
    if (parent) parents.add(parent)
  }
  return { featureContexts, parents }
}

/**
 * その段に置ける文書。モジュールは、モジュール、内部設計、機能群、機能スライスの段を持つ。
 * 機能群は子のディレクトリを持つ段であり、境界と索引だけを書く。機能スライスはモジュールから
 * 一段か二段下の、子を持たない段である。それより下と、印を持たないモジュールの段と、
 * 固定の一覧にないシステムの段には何も置けない。
 */
export function documentAllowance(directory: string, view: DocumentSetView): DocumentAllowance {
  const match = directory.match(/^docs\/domain\/([^/]+)(?:\/(.+))?$/)
  const context = match?.[1]
  if (!context) return { names: canonicalDocumentNames(directory) ?? [], freeNames: false }
  if (!view.featureContexts.has(context)) return { names: [], freeNames: false }
  const rest = match[2]?.split('/') ?? []
  if (rest.length === 0) return { names: CONTEXT_DOCUMENTS, freeNames: false }
  if (rest[0] === 'design') {
    return rest.length === 1
      ? { names: DESIGN_DOCUMENTS, freeNames: true }
      : { names: [], freeNames: false }
  }
  if (rest.length > 2) return { names: [], freeNames: false }
  if (view.parents.has(directory)) return { names: ['README.md'], freeNames: false }
  return { names: FEATURE_SLICE_DOCUMENTS, freeNames: true }
}

export function allowsDocument(allowance: DocumentAllowance, name: string): boolean {
  if (allowance.names.includes(name)) return true
  return allowance.freeNames && FREE_DOCUMENT_NAME.test(name) && !RESERVED_FREE_NAMES.has(name)
}
