namespace IdMagic.Contract {

@doc("ListAdminAuditEvents のクエリパラメータ。tenant_id は context から付与されるため\nフィールドには含めない。type は SCL events セクションの型名。")
model AuditEventQuery {
  @doc("type 完全一致フィルタ (機械向け低レベル)。例: AccessTokenIssued")
  type?: string;
  @doc("イベントカテゴリ絞り込み。authentication / success / fail / aggregated / user / group / client / consent / token / tenant / key。")
  category?: string;
  @doc("操作者 (actor) の user sub による完全一致フィルタ。filter の actor.id と同じ対象を指す\n(AuditActorVsTarget 参照)。管理操作の対象ユーザーを絞り込みたい場合は filter の target.id を使う\n(本フィールドとは別概念)。UI 上のラベルは「ユーザー ID (操作者)」。")
  user_id?: string;
  @doc("操作者 (actor) のログイン名による検索。実アカウントが常に確定するイベント\n(UserAuthenticated、OAuth2 フロー系イベント等) は payload に username を持たないため、\n検索時に UserRepo で user_id へ解決してから user_id と同じ経路で絞り込む。該当ユーザーが\n存在しない場合は 0 件を返す (フィルタ無視で全件返すフォールバックはしない)。実在しないアカウント\n名も追跡する必要がある認証失敗イベントの検索には使えない (filter の actor.username を使う)。")
  username?: string;
  @doc("occurred_at がこの時刻以降。")
  after?: utcDateTime;
  @doc("occurred_at がこの時刻以前。")
  before?: utcDateTime;
  @doc("応答の Link response header の URL に埋め込まれた値をそのまま渡す。direction・anchor・\n遷移先 page number・keyset を compact binary payload に持ち、tenant・sort・filter を associated data として\n署名した stateless versioned opaque token。新規 v3 token は時刻で失効せず、legacy v0/v2 token も既存規則で受理する。")
  cursor?: string;
  @doc("返却件数の上限 (default 100, max 1000)。")
  limit?: int64;
  @doc("raw 保存された text 列 (type / outcome 等) への部分一致フリーテキスト。PII 列は対象外。")
  q?: string;
  @doc("registry allowlist の field/operator による構造化検索式の連言 (AND)。actor.username は\n実在しないアカウント名も追跡する必要がある認証失敗イベント専用の平文一致検索で、hash 化はしない。\n実アカウントが確定するイベントの検索には username query param を使う (本フィールドとは別経路)。\nclient.ip も平文一致で照合する (truncate/hash はしない)。event.type は SCL events セクションの型名\n(owns_events) の allowlist、outcome は Go 側 eventOutcome 分類の enum (success / failure) から選ぶ\n選択式の値であり、自由入力ではない (AuditActorVsTarget / AuditSearchRegistry 参照)。actor.id /\nactor.username は操作者、target.id は管理操作の対象ユーザーを指し、意味が異なる。workflow.id /\nworkflow_run.id / workflow_step.id は IdGovernance.LifecycleWorkflow* / WorkflowRun* /\nWorkflowStepFailed イベントの workflowId/runId/stepIndex から eq 一致で検索できる registry 属性で、\nattribute value や email 本文は payload に含まないため検索対象にもならない。")
  filter?: AuditEventFilterExpression[];
}


@doc("監査イベント検索 filter 式で使える比較演算子。registry の属性ごとに許可 operator が宣言される。eq は完全一致、in は値集合のいずれか一致、contains は部分一致 (q 用)、time_range は occurred_at の期間指定。")
enum AuditEventFilterOperator {
  eq: "eq",
  in: "in",
  contains: "contains",
  time_range: "time_range",
}


@doc("検索属性を sidecar に保存 / 検索する前に平文値へ適用する変換。現状は\nnone (raw 保存) のみ存在する。")
enum AuditEventSearchTransform {
  none: "none",
}


@doc("監査イベント検索で許可される検索軸 (registry エントリ)。field/operator の allowlist を宣言し、\n任意 SQL / JSONPath ではなく閉じた検索文法を定める。Go 側の AuditSearchRegistry がこの value_object の twin。\nactor.username / client.ip を含む全属性が平文のまま sidecar に保存・検索される。actor.id /\nactor.username は操作者 (誰が行ったか)、target.id は管理操作の対象ユーザー (誰に対して行ったか) を指し、\n同じ「ユーザー」でも意味が異なる (AuditActorVsTarget)。event.type / outcome は値が自由入力ではなく\nallowlist / enum 由来の選択式として扱う: event.type は SCL events の型名 (owns_events)、outcome は\nGo 側 eventOutcome() が返す success / failure の 2 値 (該当しないイベントでは値なし)。")
model AuditEventSearchAttribute {
  @doc("検索軸の正準ドット名。例: actor.username / client.ip / event.type。")
  field: string;
  @doc("平文をそのまま sidecar に保存してよいか。現状は全属性で true。")
  raw_storable: boolean;
  @doc("保存 / 検索前に平文値へ適用する変換 (現状は none のみ)。")
  transform: AuditEventSearchTransform;
  @doc("この属性に許可される operator の allowlist。")
  allowed_operators: AuditEventFilterOperator[];
  @doc("admin 検索ビルダーの UI プリセットに出すか。")
  ui_visible: boolean;
}


@doc("AuditEventQuery.filter の 1 項。registry allowlist の field と operator、値の並びからなる連言 (AND) の 1 要素。\n任意 SQL は許さない。")
model AuditEventFilterExpression {
  @doc("registry の AuditEventSearchAttribute.field。")
  field: string;
  @doc("属性の allowed_operators に含まれる operator。")
  operator: AuditEventFilterOperator;
  @doc("operator に応じた値の並び (eq=1 / in=N / time_range=2)。PII 属性は平文入力をサーバ側で transform してから照合する。")
  values: string[];
}


@doc("管理者向けに公開する DomainEvent。payload は構造化 JSON。\n返却時に PII フィールド (email / IP 等) は EventSink 側で redact 済み前提。\nappend_only で 7 年保持する (GDPR Art. 30)。")
model AdminAuditEventResponse {
  @doc("ストア側で割り当てたイベント ID (UUIDv4)。")
  id: string;
  @doc("イベントが帰属するテナント ID。")
  tenant_id: string;
  @doc("DomainEvent.type ワイヤ名。")
  type: string;
  occurred_at: utcDateTime;
  @doc("イベント本体 (SCL events セクションの payload 形状)。")
  payload: unknown;
}


@doc("監査イベントの検索結果。ページング navigation は body ではなく Link response header で表現する。")
model AdminAuditEventListResponse {
  events: AdminAuditEventResponse[];
}


@doc("event.type / outcome を選択式にするための選択肢一覧。UI がハードコードせず、\nGo 側の単一の正 (category → type マップ / eventOutcome 分類) から機械的に取得する。")
model AuditEventSearchOptionsResponse {
  @doc("選択可能な event.type の値一覧 (SCL events の型名)。カテゴリ分類済みの type 群の和集合で、\nadmin UI のイベントカテゴリ選択肢と同じ母集合。")
  event_types: string[];
  @doc("選択可能な outcome の値一覧。現時点では success / failure の 2 値 (認証系イベントのみ分類対象)。")
  outcomes: string[];
}

}
