namespace IdMagicUi;

using TypeSpec.Reflection;

/** ページの領域。 */
enum Area { authentication, account, admin, system }

/** 利用者の作業による UI の分類。API の所有モジュールとは独立させる。 */
enum Category {
  @doc("サインイン")
  signIn: "sign-in",
  @doc("アカウント復旧")
  recovery: "recovery",
  @doc("ユーザー管理")
  users: "users",
  @doc("プロフィール") profile: "profile",
  @doc("セキュリティ") security: "security",
  @doc("アプリケーション") applications: "applications",
  @doc("同意と承認") consent: "consent",
  @doc("外部サービス接続") connections: "connections",
  @doc("API トークン") apiTokens: "api-tokens",
  @doc("グループ管理") groups: "groups",
  @doc("Agent 管理") agents: "agents",
  @doc("認可") authorization: "authorization",
  @doc("ディレクトリ連携") directories: "directories",
  @doc("フェデレーション") federation: "federation",
  @doc("設定") settings: "settings",
  @doc("テナント") tenants: "tenants",
  @doc("監査") audit: "audit",
  @doc("ジョブ") jobs: "jobs",
  @doc("鍵") keys: "keys",
  @doc("プロビジョニング") provisioning: "provisioning",
  @doc("ライフサイクル") lifecycle: "lifecycle",
  @doc("ワークロード ID") workload: "workload",
  @doc("入口と共通表示") navigation: "navigation",
}

/**
 * ページ一覧と詳細に共通する設計情報。
 * id は実装の PageMarker の kind と同じ値にし、経路と URL は実装から導く。
 * 同じ kind の経路が複数ある、または kind を動的に決める経路だけ route を書く。
 */
model PageInfo {
  id: string;
  name: string;
  area: Area;
  route?: string;
  /** 複数の異なるページが同じマーカーを使う場合だけ、route とともに明示する。 */
  marker?: string;
  actor: string;
  entry: string;
  context: string;
  purpose: string;
  content: string[];
  references: string[];
}

/**
 * 契機から結果への一つの分岐。
 * 省略時は ready から ready への無条件の移動とし、渡すデータはない。
 * mode が data の場合は result を書く。
 */
model TransitionInfo {
  event: string;
  from?: string;
  condition?: string;
  result?: string;
  toState?: string;
  data?: string;
  mode?: "navigation" | "data";
}

/** 描画しない経路も照合対象に残す。 */
model RouteInfo {
  route: string;
  kind: "layout" | "redirect" | "alias";
  reason: string;
}

/** 外部の到達先または URL のない共通表示。 */
model DestinationInfo {
  id: string;
  name: string;
  kind: "external" | "shared";
  description: string;
}

/** モジュールが担当する一つの利用目的と、その到達点。 */
model FlowInfo {
  id: string;
  name: string;
  goal: string;
  notes: string[];
}

/** 同じページでも確認や結果表示が異なれば、別の表示として扱う。 */
model FlowViewInfo {
  name?: string;
  state: string;
  visible: string[];
}

/** 表示の間を進む契機。補助的な戻り道だけを区別する。 */
model FlowLinkInfo {
  trigger: string;
  alternate?: boolean;
  /** 起点の表示のページが持つ操作の名前。生成時に存在と到達先の一致を検査する。 */
  action?: string;
}

/** モジュールの主要な利用手順。 */
auto dec flow(target: Model, info: valueof FlowInfo);
/** 利用者が見るページや外部の表示。 */
auto dec view(target: ModelProperty, info: valueof FlowViewInfo, page: ModelProperty);
/** 同じ手順内の表示をつなぎ、既存の操作を参照する。 */
auto dec flowLink(target: ModelProperty, info: valueof FlowLinkInfo, from: ModelProperty, to: ModelProperty);

/** 一覧に載せるページ。 */
auto dec page(target: ModelProperty, info: valueof PageInfo);
/** ページを探すための作業カテゴリ。 */
auto dec category(target: ModelProperty, category: valueof Category);
/** 実行する既存 API。 */
auto dec uses(target: ModelProperty, ...operations: Operation[]);
/** 操作結果から次のページまたは状態への参照。 */
auto dec transition(target: ModelProperty, info: valueof TransitionInfo, destination: ModelProperty);
/** ページとして数えない経路。 */
auto dec route(target: ModelProperty, info: valueof RouteInfo);
/** 外部の到達先または共通表示。 */
auto dec destination(target: ModelProperty, info: valueof DestinationInfo);
