namespace IdMagic.Contract {

@doc("ListAdminAuditEvents のクエリパラメーター。`tenant_id` はコンテキストから得るので、ここのフィールドではない。`type` は `DomainEventEnvelope.type`、すなわち発行する Context がイベントに付けた種類の名前である。")
model AuditEventQuery {
  @doc("`type` の完全一致。機械向けの低水準の絞り込みである。たとえば `AccessTokenIssued`。")
  type?: string;
  @doc("イベントの分類で絞り込む。authentication、success、fail、aggregated、user、group、client、consent、token、tenant、key のいずれか。")
  category?: string;
  @doc("操作した主体のユーザーの sub の完全一致。`filter` の `actor.id` と同じものを指す（AuditActorVsTarget を参照）。管理操作の対象のユーザーで絞り込むときは、このフィールドとは別の概念である `filter` の `target.id` を使う。UI では「ユーザー ID（操作した主体）」と表示する。")
  user_id?: string;
  @doc("操作した主体のログイン名で検索する。実在のアカウントが常に確定しているイベント（UserAuthenticated、OAuth2 のフローのイベントなど）はペイロードにユーザー名を含まないので、検索は UserRepo で名前をユーザー ID へ解決し、`user_id` と同じく完全一致で絞り込む。該当するユーザーが存在しないときは結果を空にし、絞り込みを無視してすべてを返すことはない。存在しなかったアカウント名も追跡する必要がある認証の失敗の検索には使えないので、それには `filter` の `actor.username` を使う。")
  username?: string;
  @doc("`occurred_at` がこの時点以後。")
  after?: utcDateTime;
  @doc("`occurred_at` がこの時点以前。")
  before?: utcDateTime;
  @doc("レスポンスの Link ヘッダーの URL に埋め込まれた値をそのまま渡す。方向、基準点、移動先のページ番号、キーセットを小さなバイナリのペイロードに収めた、状態を持たない、バージョン付きの不透明なトークンで、テナント、並び順、絞り込み条件を関連データとして署名する。新しい v3 のトークンは時間で期限切れにならず、従来の v0 と v2 のトークンもそれぞれの既存の規則で受け付ける。")
  cursor?: string;
  @doc("ページの大きさの上限（デフォルトは 100、最大は 1000）。")
  limit?: int64;
  @doc("生のまま保存したテキストの列（`type`、`outcome` など）に部分一致させる自由入力のテキスト。個人情報の列は対象外である。")
  q?: string;
  @doc("レジストリのフィールドと演算子の許可リストに対する、構造化された検索条件の論理積（AND）。`actor.username` は認証の失敗に限った平文の一致で、存在しなかったアカウント名も追跡できるようにするためのものであり、ハッシュ化しない。実在のアカウントが確定しているイベントを検索するときは、このフィールドとは別の経路である `username` のクエリパラメーターを使う。`client.ip` も平文で照合し、切り詰めもハッシュ化もしない。`event.type` は Context が発行する `DomainEventEnvelope.type` の名前から、`outcome` は Go の `eventOutcome` の分類が定める列挙（success または failure）から選び、どちらも自由入力ではなく一覧から選ぶ（AuditActorVsTarget と AuditSearchRegistry を参照）。`actor.id` と `actor.username` は操作した主体を、`target.id` は管理操作の対象のユーザーを指し、意味が異なる。`workflow.id`、`workflow_run.id`、`workflow_step.id` はレジストリの属性で、IdGovernance の LifecycleWorkflow*、WorkflowRun*、WorkflowStepFailed のイベントの workflowId、runId、stepIndex に `eq` で照合する。`actor.type`（`user` または `agent`）、`agent.id`、`delegation.actor`、`delegation.depth`、`delegation.mode` は委任の軸である。`delegation.actor` はイベントの `act` の連鎖のすべての主体を指し、どれか一つが一致すれば一致するので、一つの絞り込み条件でどの参加者からも連鎖全体を見つけられる。`delegation.mode` は、REQ-OAUTH2-049 が定める導出がペイロードに書いた値を読み、監査が改めて導くことはしない。これらの軸ができる前に記録したイベントはどの軸の値も含まないので、これらの軸のどの値にも一致しない。属性値とメールの本文はペイロードに含まれないので、検索もできない。")
  filter?: AuditEventFilterExpression[];
}


@doc("監査の検索の絞り込み式で使える比較演算子。各属性で使える演算子はレジストリが宣言する。`eq` は完全一致、`in` は値の集合のいずれかとの一致、`contains` は部分一致（`q` が使う）、`time_range` は `occurred_at` の範囲の指定である。")
enum AuditEventFilterOperator {
  eq: "eq",
  in: "in",
  contains: "contains",
  time_range: "time_range",
}


@doc("検索属性をサイドカーに保存または検索する前に、平文の値へ施す変換。現在は値を生のまま保存する `none` だけである。")
enum AuditEventSearchTransform {
  none: "none",
}


@doc("監査イベントの検索で許可する検索軸、すなわちレジストリの一項目。フィールドと演算子の許可リストを宣言することで、任意の SQL や JSONPath の代わりに閉じた検索の文法を定める。Go の AuditSearchRegistry はこの値オブジェクトと対になる。`actor.username` と `client.ip` を含め、どの属性もサイドカーに平文で保存し、平文で検索する。`actor.id` と `actor.username` は操作した主体（誰がしたか）を、`target.id` は管理操作の対象のユーザー（誰に対してしたか）を指し、同じ「ユーザー」という語が別のものを指す（AuditActorVsTarget）。`event.type` と `outcome` は自由入力ではなく一覧から選ぶ。`event.type` は Context が発行する `DomainEventEnvelope.type` の名前から、`outcome` は Go の `eventOutcome()` が返す success と failure の二値から選び、分類しないイベントには値がない。`actor.type`、`agent.id`、`delegation.actor`、`delegation.depth`、`delegation.mode` は委任の軸であり（REQ-AUDIT-005、REQ-AUDIT-006）、Agent がユーザーのために行ったことと、ユーザー自身が行ったことを区別する（AuditDelegationChain）。")
model AuditEventSearchAttribute {
  @doc("検索軸の、ドットで区切った正式な名前。`actor.username`、`client.ip`、`event.type` など。")
  field: string;
  @doc("平文をそのままサイドカーに保存してよいかどうか。現在はすべての属性で true である。")
  raw_storable: boolean;
  @doc("保存または検索の前に平文へ施す変換。現在は `none` だけである。")
  transform: AuditEventSearchTransform;
  @doc("この属性で使える演算子の許可リスト。")
  allowed_operators: AuditEventFilterOperator[];
  @doc("一つのイベントがこの軸に複数の値を含んでよいかどうか。現在、複数値の軸は `delegation.actor` だけで、一つのイベントが `act` の連鎖のすべての主体を指し、どれか一つが一致すれば絞り込みに一致する。それ以外の軸は、一つのイベントにつき値を一つまでしか含まない。")
  multi_valued: boolean;
  @doc("管理用の検索ビルダーで、UI のプリセットとして示すかどうか。")
  ui_visible: boolean;
}


@doc("`AuditEventQuery.filter` の一項。レジストリの許可リストにあるフィールドと演算子に値を組み合わせたもので、論理積（AND）の一要素である。任意の SQL は許可しない。")
model AuditEventFilterExpression {
  @doc("レジストリにある AuditEventSearchAttribute の `field`。")
  field: string;
  @doc("属性の `allowed_operators` に載っている演算子。")
  operator: AuditEventFilterOperator;
  @doc("演算子がとる数の値。`eq` では一つ、`in` では N 個、`time_range` では二つである。個人情報の属性では、サーバーが照合の前に平文の入力を変換する。")
  values: string[];
}


@doc("管理者に公開する DomainEvent。ペイロードは構造化された JSON である。メールアドレスや IP のような個人情報のフィールドは、この時点までに EventSink が秘匿している前提である。ストアは追記専用で、イベントを 7 年間保持する（GDPR 第 30 条）。")
model AdminAuditEventResponse {
  @doc("ストアが割り当てたイベントの id（UUIDv4）。")
  id: string;
  @doc("イベントが属するテナント。")
  tenant_id: string;
  @doc("`DomainEvent.type` の送受信上の名前。")
  type: string;
  occurred_at: utcDateTime;
  @doc("DomainEventPayload の形のイベントの本体。公開フィールドが契約で、それ以外は発行する Context に属する。")
  payload: Record<unknown>;
}


@doc("監査イベントの検索の結果。ページングの移動先はボディではなく Link レスポンスヘッダーにある。")
model AdminAuditEventListResponse {
  events: AdminAuditEventResponse[];
}


@doc("列挙型の検索軸を選択肢の一覧にするための値。UI はこれを直接書かず、Go の同じ唯一の情報源（分類から種類への対応表、eventOutcome の分類、REQ-OAUTH2-049 が定める DelegationMode の導出）から読む。")
model AuditEventSearchOptionsResponse {
  @doc("選べる `event.type` の値、すなわち `DomainEventEnvelope.type` の名前。分類済みの種類の和集合であり、管理 UI のイベントの分類の選択肢と同じ母集団である。")
  event_types: string[];
  @doc("選べる `outcome` の値。現在は success と failure の二つで、分類するのは認証のイベントだけである。")
  outcomes: string[];
  @doc("選べる `actor.type` の値。`user` と `agent` である。")
  actor_types: string[];
  @doc("選べる `delegation.mode` の値。DelegationMode の要素なので、監査の絞り込みとイントロスペクションのレスポンスが同じモードを指す。")
  delegation_modes: string[];
}


@doc("リクエストが指定した監査イベントがこのテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:event_not_found` である。")
model AuditEventNotFoundError is ProblemDetails;

}
