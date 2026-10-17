export interface DirectoryListing {
  directory: string
  files: string[]
}

export const ROOT_DOCUMENTS = ['README.md'] as const

/** モジュール共通の語彙。各モジュールの `glossary.md` はここの語を狭めて使う。 */
export const SHARED_GLOSSARY = 'docs/modules/glossary.md'

/** 複数のモジュールの協調による保証と、複数のモジュールへ共通に適用する保証を宣言する文書。 */
export const SHARED_SCENARIOS = 'docs/modules/scenarios.feature.md'

/**
 * モジュール設計文書の入口。直下にはこの索引、モジュール共通の文書、モジュールのディレクトリだけを置く。
 * 共通の文書はモジュールでも機能スライスでもない。
 */
export const MODULE_INDEX_DOCUMENTS = ['README.md', 'glossary.md', 'scenarios.feature.md'] as const

/**
 * 全体設計文書の設計領域。システムの `docs/design/<directory>/` とモジュールの直下の
 * `<file>` が同じ語彙と順序を使う。モジュールの索引は固有設計と例外だけを並べる。
 * 順序は実現対象（アプリケーション、データ、
 * インフラストラクチャ）、複数の実現対象へ作用する領域、検証の順であり、索引の表と生成サイトの
 * 目次はこの順に従う。
 */
export const DESIGN_AREAS = [
  { name: 'アーキテクチャ設計', directory: 'architecture', file: 'architecture.md' },
  { name: 'アプリケーション設計', directory: 'application', file: 'application.md' },
  { name: 'データ設計', directory: 'data', file: 'data.md' },
  { name: 'インフラストラクチャ設計', directory: 'infrastructure', file: 'infrastructure.md' },
  { name: 'セキュリティ設計', directory: 'security', file: 'security.md' },
  { name: '信頼性設計', directory: 'reliability', file: 'reliability.md' },
  { name: '性能設計', directory: 'performance', file: 'performance.md' },
  { name: 'オブザーバビリティ設計', directory: 'observability', file: 'observability.md' },
  { name: '検証設計', directory: 'verification', file: 'verification.md' },
] as const

/** 設計領域ごとの固定の文書。`DESIGN_AREAS` の順に並べて一次情報文書の一覧へ入れる。 */
const DESIGN_AREA_DOCUMENTS: Record<(typeof DESIGN_AREAS)[number]['directory'], readonly string[]> =
  {
    architecture: [
      'README.md',
      'system-boundary.md',
      'logical.md',
      'runtime.md',
      'deployment.md',
      'decisions.md',
      'risks.md',
    ],
    application: [
      'README.md',
      'api-guidelines.md',
      'design-guidelines.md',
      'backend.md',
      'frontend.md',
      'user-interface.md',
    ],
    data: ['README.md', 'database.md', 'schema-management.md', 'lifecycle.md'],
    infrastructure: ['README.md', 'platform.md', 'network.md'],
    security: ['README.md', 'threat-model.md', 'authorization.md', 'secrets.md'],
    reliability: ['README.md', 'availability.md', 'recovery.md'],
    performance: ['README.md', 'capacity.md', 'scaling.md'],
    observability: ['README.md', 'monitoring.md', 'logging.md', 'tracing.md'],
    verification: ['README.md', 'system-acceptance.md', 'security.md'],
  }

/** 一次情報文書を、`docs/README.md` の区分の順に、上位から読む順序で定義する。 */
export const SYSTEM_DOCUMENT_DIRECTORIES: ReadonlyArray<{
  directory: string
  names: readonly string[]
}> = [
  { directory: 'docs', names: ROOT_DOCUMENTS },
  {
    directory: 'docs/requirements',
    names: ['README.md', 'product-overview.md', 'functional.md', 'quality.md', 'standards.md'],
  },
  { directory: 'docs/design', names: ['README.md'] },
  ...DESIGN_AREAS.map(({ directory }) => ({
    directory: `docs/design/${directory}`,
    names: DESIGN_AREA_DOCUMENTS[directory],
  })),
  { directory: 'docs/modules', names: MODULE_INDEX_DOCUMENTS },
  {
    directory: 'docs/operations',
    names: ['README.md', 'service-management.md', 'maintenance.md'],
  },
  {
    directory: 'docs/formats',
    names: [
      'README.md',
      'documentation-guide.md',
      'specification-format.md',
      'design-format.md',
      'work-item-format.md',
    ],
  },
]

const SYSTEM_DOCUMENTS_BY_DIRECTORY = new Map<string, readonly string[]>(
  SYSTEM_DOCUMENT_DIRECTORIES.map(({ directory, names }) => [directory, names]),
)

export const SYSTEM_DOCUMENT_PATHS = SYSTEM_DOCUMENT_DIRECTORIES.flatMap(({ directory, names }) =>
  names.map((name) => `${directory}/${name}`),
)

/**
 * 内容に応じた任意名を許し、閉じたファイル集合の対象にしない段。`docs/modules` は
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
 * モジュールの直下に置ける固定の文書。文書は対象（モジュール全体、設計領域、機能）で分け、
 * 仕様か設計かでは分けない。並びは索引と生成サイトの順であり、概要、語彙、外部規範、品質、
 * 設計領域、設計の記録の順とする。`quality.md` は、システムの品質要件のうちこのモジュールに
 * 割り当てた分を書く。モジュールの直下には、この一覧のほかに文書を置かない。
 */
export const MODULE_DOCUMENTS: readonly string[] = [
  'README.md',
  'glossary.md',
  'standards.md',
  'quality.md',
  ...DESIGN_AREAS.map(({ file }) => file),
  'decisions.md',
  'risks.md',
]

/** 機能仕様を置く、モジュールの下の唯一の段。 */
export const FEATURES_DIRECTORY = 'features'

/**
 * 機能名と章の名前に使えない名前。ファイル種別ごとの文書を機能や章として置くと、一つの対象を
 * 種別ごとのファイルに散らす構造へ戻ってしまう。用語、標準、品質、設計の記録はモジュールの
 * 直下に置き、機能の設計は機能仕様の節に置くので、`design` も名前にしない。
 */
const RESERVED_FEATURE_STEMS = new Set([
  'states',
  'internals',
  'design',
  'features',
  'scenarios',
  'glossary',
  'standards',
  'quality',
  'decisions',
  'risks',
  'readme',
])

/**
 * 機能の段の文書の名前。`<feature>.md` が機能仕様、`<feature>.feature.md` が受け入れ例、
 * `<feature>.<chapter>.md` が長くなった機能仕様の章である。名前はケバブケースに限り、
 * `Readme.md` のような打ち間違いを通さない。
 */
const KEBAB = '[a-z0-9]+(?:-[a-z0-9]+)*'
const FEATURE_DOCUMENT_NAME = new RegExp(`^(${KEBAB})(?:\\.(${KEBAB}))?\\.md$`)

/** 機能の段の文書が、機能のどの部分か。 */
export type FeatureDocument = {
  module: string
  feature: string
  part: 'specification' | 'chapter' | 'examples'
}

/** パスが機能の段の文書なら、その機能と部分を返す。名前の形が合わなければ返さない。 */
export function featureDocument(path: string): FeatureDocument | undefined {
  const match = path.match(/^docs\/modules\/([^/]+)\/features\/([^/]+)$/)
  if (!match) return undefined
  const name = match[2]?.match(FEATURE_DOCUMENT_NAME)
  const [module = '', feature = '', chapter] = [match[1], name?.[1], name?.[2]]
  if (!name || RESERVED_FEATURE_STEMS.has(feature) || RESERVED_FEATURE_STEMS.has(chapter ?? '')) {
    return undefined
  }
  const part =
    chapter === undefined ? 'specification' : chapter === 'feature' ? 'examples' : 'chapter'
  return { module, feature, part }
}

/** 機能仕様のパス。機能の章と例はこのパスと同じ機能に属する。 */
export function featureSpecificationPath(module: string, feature: string): string {
  return `docs/modules/${module}/${FEATURES_DIRECTORY}/${feature}.md`
}

/**
 * ある段に置ける文書。`features` が真なら、固定の名前の代わりに機能の段の名前の形で置ける。
 */
export interface DocumentAllowance {
  names: readonly string[]
  features?: boolean
}

/** 段の集合から読み取った、名前の判定に要る事実。 */
export interface DocumentSetView {
  /**
   * `README.md` を持つモジュールの名前。この印のないモジュールの段には、どの文書も置けない。
   * 設定を読まずに同じ判定ができるよう、ファイルの有無で決める。
   */
  featureModules: ReadonlySet<string>
}

export function describeDocumentSet(listings: readonly DirectoryListing[]): DocumentSetView {
  const featureModules = new Set<string>()
  for (const listing of listings) {
    const module = listing.directory.match(/^docs\/modules\/([^/]+)$/)?.[1]
    if (module && listing.files.includes('README.md')) featureModules.add(module)
  }
  return { featureModules }
}

/**
 * その段に置ける文書。モジュールは、モジュールの直下と機能の段の二つだけを持つ。
 * それより下と、印を持たないモジュールの段と、固定の一覧にないシステムの段には何も置けない。
 */
export function documentAllowance(directory: string, view: DocumentSetView): DocumentAllowance {
  const match = directory.match(/^docs\/modules\/([^/]+)(?:\/(.+))?$/)
  const module = match?.[1]
  if (!module) return { names: canonicalDocumentNames(directory) ?? [] }
  if (!view.featureModules.has(module)) return { names: [] }
  const rest = match[2]
  if (rest === undefined) return { names: MODULE_DOCUMENTS }
  if (rest === FEATURES_DIRECTORY) return { names: [], features: true }
  return { names: [] }
}

export function allowsDocument(allowance: DocumentAllowance, name: string): boolean {
  if (allowance.names.includes(name)) return true
  return (
    allowance.features === true &&
    featureDocument(`docs/modules/m/${FEATURES_DIRECTORY}/${name}`) !== undefined
  )
}
