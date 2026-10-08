/**
 * ドメインイベントの公開項目の語彙が、TypeSpec の宣言と Go の読み取り点で一致することを
 * 確かめる。
 *
 * イベントの payload は `AdminAuditEventResponse.payload` として不透明なまま公開されるので、
 * 契約になるのは payload の全体ではなく、Context をまたいで名前で読まれる項目だけである。
 * その名前を変えるとコンパイルは通ったまま監査の検索軸が黙って空になり、通知の宛先が
 * 解決できなくなる。片側だけを直せる状態を残さないために、両側から名前を集めて突き合わせる。
 *
 * 後方互換の判定は行わない。公開イベントの消費者はこのリポジトリの中にしかおらず、
 * 外部の消費者がいない契約にリリースベースラインを敷いても守る相手がいないからである。
 */

/** 宣言と実際の読み取りの差。両側とも安定した並びで返す。 */
export interface VocabularyDiff {
  /** Go が読んでいるのに TypeSpec が宣言していない項目。 */
  missing: string[]
  /** TypeSpec が宣言しているのに Go のどこも読んでいない項目。 */
  undeclared: string[]
}

/** 公開項目を宣言するモデルの名前。この名前だけを一次情報として読む。 */
const PAYLOAD_MODEL = 'DomainEventPayload'

/** `name: type;` と `name?: type;` の左辺。装飾子の行とスプレッドは対象にしない。 */
const PROPERTY = /^\s*([A-Za-z_][A-Za-z0-9_]*)\??\s*:/

/**
 * payload の項目名を文字列リテラルで名指しする Go の読み取り点。監査の抽出器が使う
 * 3 つのアクセサ、通知のディスパッチャーが使う `stringField` と添字、そしてカタログが
 * 宛先の項目名を持つ `RecipientField` である。
 */
const ACCESSORS = [
  /\bpayload(?:String|Strings|NumberString)\s*\([^,()]+,\s*"([^"]+)"\s*\)/g,
  /\bstringField\s*\(\s*payload\s*,\s*"([^"]+)"\s*\)/g,
  /\bpayload\[\s*"([^"]+)"\s*\]/g,
  /\bRecipientField\s*:\s*"([^"]+)"/g,
]

/**
 * TypeSpec の宣言から公開項目の名前を集める。TypeSpec として解析せず行を追うのは、
 * 読みたいのが 1 つのモデルの property 名だけであり、コンパイラを持ち込むと検査が
 * `compile-spec` の成否に連鎖するためである。
 */
export function collectDeclaredEventFields(source: string): Set<string> {
  const fields = new Set<string>()
  let inside = false
  for (const line of source.split('\n')) {
    if (!inside) {
      if (new RegExp(`^\\s*model\\s+${PAYLOAD_MODEL}\\s*\\{`).test(line)) inside = true
      continue
    }
    if (line.includes('}')) break
    const property = line.match(PROPERTY)
    if (property?.[1]) fields.add(property[1])
  }
  return fields
}

/** Go のソースから、payload の項目名を文字列で名指ししている箇所を集める。 */
export function collectConsumedEventFields(sources: Map<string, string>): Set<string> {
  const fields = new Set<string>()
  for (const source of sources.values()) {
    for (const accessor of ACCESSORS) {
      for (const match of source.matchAll(accessor)) {
        if (match[1]) fields.add(match[1])
      }
    }
  }
  return fields
}

/** 宣言と読み取りの差を、両側とも並びを固定して返す。 */
export function diffEventFieldVocabulary(
  declared: Set<string>,
  consumed: Set<string>,
): VocabularyDiff {
  const sorted = (values: Iterable<string>) => [...values].sort()
  return {
    missing: sorted([...consumed].filter((field) => !declared.has(field))),
    undeclared: sorted([...declared].filter((field) => !consumed.has(field))),
  }
}

/**
 * `EventType()` を実装する Go の型。型名と、`EventType()` が返す公開名の組で返す。
 * 発行は型の値を作ることでしか起きないので、照合には型名を使い、報告には公開名を使う。
 */
const EVENT_DECLARATION =
  /func\s*\(\s*\w+\s+\*?([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*EventType\(\)\s+string\s*\{\s*return\s+"([A-Za-z_][A-Za-z0-9_]*)"/g

/** 合成リテラル `T{` の型名。`&domain.T{` も `T{` の部分で一致する。 */
const COMPOSITE_LITERAL = /(?<![A-Za-z0-9_])([A-Z][A-Za-z0-9_]*)\{/g

export interface DeclaredEvent {
  typeName: string
  eventType: string
}

/** Go のソースから、`EventType()` を実装するイベント型を集める。 */
export function collectDeclaredEvents(source: string): DeclaredEvent[] {
  return [...source.matchAll(EVENT_DECLARATION)].map((match) => ({
    typeName: match[1] ?? '',
    eventType: match[2] ?? '',
  }))
}

/** Go のソースから、合成リテラルで値を作っている型名を集める。 */
export function collectConstructedTypes(source: string): Set<string> {
  const names = new Set<string>()
  for (const match of source.matchAll(COMPOSITE_LITERAL)) {
    if (match[1]) names.add(match[1])
  }
  return names
}

/**
 * 宣言だけがあり、本番コードのどこも値を作らないイベントの公開名を、並びを固定して返す。
 * 値を作らない型は発行されないので、監査ログを読む側からは操作が一度も起きていないことと
 * 区別できない。テストだけが作る型も発行経路を持たないので、呼び出し側はテストを渡さない。
 */
export function findEventsWithoutEmission(
  declared: readonly DeclaredEvent[],
  constructed: ReadonlySet<string>,
): string[] {
  return [
    ...new Set(
      declared.filter((event) => !constructed.has(event.typeName)).map((event) => event.eventType),
    ),
  ].sort()
}
