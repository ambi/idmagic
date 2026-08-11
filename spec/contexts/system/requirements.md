# System Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-SYSTEM-001: Operatorは分離された運用資産でSLOを検証する
- Actor: Operator
- Given: API、UI gateway、event relay は個別の実行単位として配備される
- Given: MetricsExposition の公開範囲は management network に制限される
- Then: Operator が環境 overlay を選んで運用 manifest を適用する
- Then: API の liveness、readiness、startup probe は各々 LivenessProbe、ReadinessProbe、StartupProbe を呼ぶ
- Then: Prometheus が MetricsExposition をスクレイプし、OAuth2 の availability、latency、error-rate objectives を表示・評価する
- Alternative (PostgreSQL へ到達できない): ReadinessProbe は unavailable を返し、API は新規トラフィックを受けない → LivenessProbe は healthy を維持し、依存障害だけで再起動しない
- Alternative (Prometheus Operator が導入されていない): ServiceMonitor は適用対象から外し、標準 Prometheus scrape 設定で MetricsExposition を収集する

### REQ-SYSTEM-002: orchestration probeはprocess lifecycleと依存状態を区別する
- Actor: Operator
- Then: Operator が初期化完了後の liveness、readiness、startup probe を呼ぶ
- Then: すべて 200 と healthy を返す
- Alternative (初期化中または graceful drain 中である): liveness は 200 healthy を維持する → readiness または startup は 503 を返す
- Alternative (構成された永続化依存へ到達できない): readiness は 503 unavailable を返す → liveness は 200 healthy を維持する

### REQ-SYSTEM-003: 明示的に選択した表示言語でホスト認証画面が描画される
- Actor: EndUser
- Given: 未認証セッションで Login 画面を表示している
- Then: EndUser が表示言語 "en" を選択する
- Then: Login 画面の文言が en 辞書で表示される
- Then: 選択した Locale がブラウザに保存され、以後のアクセスで保存済み設定として優先される

### REQ-SYSTEM-004: 未対応localeは既定localeにフォールバックする
- Actor: EndUser
- Given: ブラウザの言語設定が "fr" である
- Given: 表示言語の明示選択も保存済み設定も存在しない
- Then: EndUser が Login 画面を表示する
- Then: 画面の文言は既定 locale "en" の辞書で表示される

### REQ-SYSTEM-005: 起動時設定の既定localeがフォールバックに使われる
- Actor: Operator
- Given: 表示言語の明示選択、ui_locales ヒント、保存済み設定、対応ブラウザ言語が存在しない
- Then: Operator が VITE_DEFAULT_LOCALE を "ja" に設定してアプリケーションを起動する
- Then: EndUser が画面を表示する
- Then: 画面の文言は ja 辞書で表示される
- Alternative (VITE_DEFAULT_LOCALE が未設定または未対応値である): 画面の文言は FallbackLocale "en" の辞書で表示される

### REQ-SYSTEM-006: 起動時設定でVite dev server以外でもDemoLoginAffordanceが表示される
- Actor: Operator
- Given: Vite dev server ではなくビルド済み frontend を配備している
- Then: Operator が VITE_DEMO_LOGIN_ENABLED を "true" に設定してビルドする
- Then: EndUser が HomePage を表示する
- Then: HomePage は DemoLoginAffordance を表示する
- Then: EndUser が DemoLoginAffordance を選択する
- Then: development profile が seed した demo user の資格情報で authorization_code フローが完了する
- Alternative (VITE_DEMO_LOGIN_ENABLED が未設定または "true" 以外である): HomePage は DemoLoginAffordance を表示しない
- Alternative (development profile が seed されていない): authorization は既知のデモ資格情報が存在せず失敗する

### REQ-SYSTEM-007: Vite dev server実行時は設定なしでDemoLoginAffordanceが表示される
- Actor: EndUser
- Given: Vite dev server で frontend を実行している
- Then: EndUser が HomePage を表示する
- Then: VITE_DEMO_LOGIN_ENABLED の設定に関わらず HomePage は DemoLoginAffordance を表示する

### REQ-SYSTEM-008: OIDC ui_localesヒントにより表示言語が決まる
- Actor: ResourceOwner
- Given: 未認証セッションで表示言語の明示選択も保存済み設定も存在しない
- Then: "web-app" として ui_locales "en" で認可リクエストを送信する
- Then: Login 画面の文言は en 辞書で表示される
- Alternative (表示言語が既に明示選択済みの場合は ui_locales ヒントで上書きされない): EndUser が表示言語 "ja" を明示的に選択済みである → "web-app" として ui_locales "en" で認可リクエストを送信する → Login 画面の文言は ja 辞書で表示される

### REQ-SYSTEM-009: 管理者が選択した表示言語で管理画面が表示される
- Actor: Administrator
- Given: roles に "admin" を持つ Administrator が認証済みで AdminDashboard を表示している
- Then: Administrator が表示言語 "en" を選択する
- Then: AdminDashboard の文言が en 辞書で表示される

### REQ-SYSTEM-010: 選択した表示言語で全UI画面が描画される
- Actor: EndUser
- Given: 対応する画面へ遷移できる認証状態である
- Then: EndUser または Administrator が表示言語 "en" を選択する
- Then: 任意のUI画面を表示する
- Then: 画面、shared shell、dialog、empty state、aria label、状態ラベルがen辞書で表示される
- Then: 日時および数値がenの書式で表示される
- Alternative (jaを選択する): 同じ要素がja辞書およびjaの書式で表示される
- Alternative (翻訳keyが欠落している): FallbackLocale (en) の対応keyを表示する

### REQ-SYSTEM-011: 既知のバックエンドエラーコードはUIで翻訳される
- Actor: EndUser
- Given: UI操作に対しバックエンドがエラー応答を返す
- Then: バックエンドが既知のstable error codeを返す
- Then: UIが選択済みDisplayLanguageの辞書にあるerror codeの文言を表示する
- Alternative (error codeが未知、またはbackendが任意のmessageかProblem Detailsだけを返す): UIはバックエンドのmessage、error_description、detail、titleのうち利用可能な人間可読文を英語のまま表示する → バックエンドから有効なエラー応答を受信した場合は通信障害用fallbackを表示しない
- Alternative (RFC 9457 Problem Detailsのtypeが既知のstable error codeを表す): UIはtypeのurn:idmagic:error: suffixをerror codeとして解釈する → UIが選択済みDisplayLanguageの辞書にあるerror codeの文言を表示する

### REQ-SYSTEM-012: PostgreSQLクエリの期限は結果読取完了まで維持される
- Actor: Operator
- Given: PostgreSQL persistenceとquery timeoutが構成されている
- Then: Systemが共通persistence adapterで単一行または複数行queryを開始する
- Then: queryがRowまたはRowsを返す
- Then: 呼び出し側が期限内にScanまたはiterationを完了する
- Then: 結果がcontext canceledにならず返され、connectionが解放される
- Alternative (結果読取中にquery timeoutの期限へ到達する): 読取はdeadline exceededで中断される → 結果をcloseするとconnectionとtimeout resourceが解放される
- Alternative (単一行queryに該当する行が存在しない): Scanはno rowsを返す → no rowsは正常なquery応答として扱われcircuit breakerの失敗率を増加させない

### REQ-SYSTEM-013: バックエンドAPIエラーは英語で返る
- Actor: APIConsumer
- Then: APIConsumer が不正な JSON を HTTP API に送信する
- Then: System は既存の error code と HTTP status を返す
- Then: System は英語の message を返す
- Alternative (OAuth/OIDC redirect endpoint が要求を拒否する): System は既存の OAuth error code を返す → System は英語の error_description を返す
- Alternative (未知の内部エラーが発生する): System は既存の error code と HTTP status を維持する → System は英語のエラー本文を返す

### REQ-SYSTEM-014: 非推奨のinterfaceを呼ぶとDeprecation/Sunsetヘッダが返る
- Actor: APIConsumer
- Given: stable な interface に deprecated_since が設定されている
- Then: APIConsumer が非推奨マークされた interface を呼び出す
- Then: 応答に Deprecation ヘッダが付与される
- Alternative (interface に sunset_at も設定されている): 応答に Sunset ヘッダも付与される
- Alternative (interface が deprecated_since を設定していない): 応答に Deprecation ヘッダは付与されない

### REQ-SYSTEM-015: 管理コンソールとアカウントポータルは失効セッションから同一画面に復帰する
- Actor: Administrator
- Given: Administrator が first-party の管理コンソールで access token を保持している
- Given: 保持している access token が失効している
- Then: Administrator が AdminDashboard で管理 API を呼び出す
- Then: API が 401 を返す
- Then: 保持していた access token / refresh token と OIDC callback state を破棄する
- Then: 直前の画面への同一オリジン相対 return_to を保ったまま再認可を1回だけ開始する
- Then: 再ログイン完了後に元の AdminDashboard へ復帰する
- Alternative (再認可から復旧できない): 再ログイン導線を提示する

### REQ-SYSTEM-016: LivenessProbe
プロセスが要求を処理できることを示す liveness probe。依存サービスの一時障害では失敗させない。

### REQ-SYSTEM-017: ReadinessProbe
起動完了、drain 状態、構成された永続化依存の到達性を確認する readiness probe。verbose 指定時だけ依存詳細を返す。

### REQ-SYSTEM-018: StartupProbe
初期化完了までは 503 starting、完了後は 200 healthy を返す startup probe。

### REQ-SYSTEM-019: BackendErrorResponse
バックエンドの HTTP API とプロトコル endpoint がエラー時に返す共通の外部契約。
message、error_description、detail、またはプレーンテキスト本文のいずれも
BackendErrorText であり、表示言語 (DisplayLanguage) に関わらず常に英語で固定する。
RFC 9457 Problem Details では type の urn:idmagic:error: suffix を stable error code として
解釈でき、UI は detail または title を人間可読な fallback として利用できる。
個別 endpoint の error code と HTTP status はこの契約によって変更しない。
- Postcondition: text_is_english(output.message)
- Postcondition: text_is_english(output.error_description)
- Postcondition: text_is_english(output.detail)

### REQ-SYSTEM-020: MetricsExposition
Prometheus/OpenMetrics 形式でスクレイプ可能な集約メトリクスを公開する management endpoint。
idmagic-api は HTTP RED (リクエスト数・エラー率・レイテンシ) と認証ドメインのゴールデンシグナル
(login 成否、token 発行、throttle ヒット) を、idmagic-worker は Jobs の実行レーン別ゴールデンシグナル
(queue depth、claim latency、成功・失敗、retry。lane は ExecutionLane の有限集合) を公開する
independent な endpoint インスタンスであり、両者は同一プロセスを共有しない。tenant_id、user_id、
client_id、job_id、path 実値など高カーディナリティな値は label に含めず、route template や
outcome class、lane などの有限集合だけを label とする。application API の tenant middleware から
分離し、deploy policy で loopback/management network または認証のいずれかにより公開範囲を制限する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Locale | UI表示言語を一意に決めるBCP47言語タグ。idmagicは "ja" と "en" のみをサポート対象とし、それ以外は未対応 locale として扱う。 | locale tag, 表示言語コード |
| DisplayLanguage | EndUser または Administrator が言語切り替え UI で明示的に選択した Locale。選択はブラウザに保存され、以後のアクセスで保存済み設定として優先される。 | 表示言語, 言語設定 |
| FallbackLocale | 要求された Locale が未対応、または対応 Locale の辞書に該当 translation key が欠落している場合に表示へ用いる既定 Locale。idmagicでは "en" を既定とする。 | 既定 locale, default locale |
| ConfiguredDefaultLocale | アプリケーション起動時の設定 VITE_DEFAULT_LOCALE により指定する既定 Locale。"ja" または "en" のみを受け付け、未設定または未対応値のときは FallbackLocale を使う。 | startup default locale, configured locale fallback |
| DemoLoginAffordance | HomePage が表示する、ローカルデモ資格情報 (Seeding の development profile が作成する demo user と demo OAuth2 client) を使った authorization_code フローへの近道。Vite dev server 実行時は既定で表示し、それ以外のビルドではアプリケーション起動時の設定 VITE_DEMO_LOGIN_ENABLED を "true" に明示したときだけ表示する。表示条件は development profile が実際に seed 済みかどうかを問わない。 | demo login shortcut, ローカルデモ認証の近道 |
| BackendErrorText | バックエンドが HTTP、OAuth/OIDC redirect、SAML、SCIM などの外部 API 応答で返す利用者向けエラー本文。message、error_description、detail およびプレーンテキストのエラー本文を含む。常に英語であり、表示言語によって変化しない。 | API error message, error description |
| PersistedStateModel | created_at を持ち、作成後に現在状態が更新される場合は updated_at も持つ永続化状態モデルの規約。作成後は不可変で消費・削除のみされる記録モデルは updated_at を持たない。issued_at / granted_at / occurred_at / expires_at / revoked_at などのドメイン時刻は created_at を置き換えない。各 context のモデル定義はこの規約に従う。 |  |
| EndUser | 認証済みまたは認証を試みる一般利用者。 |  |
| Operator | IdP をデプロイ・起動時設定を行う運用者。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |
| Administrator | テナント内または横断のリソースを管理する権限を持つ利用者。 |  |
| APIConsumer | HTTP API を直接呼び出す外部クライアント。 |  |
| InterfaceStability | interface の外部契約としての性質を表す区分。stable は互換を保証する外部契約、beta は互換保証前の外部契約、internal は browser session 専用または domain-internal で外部契約に含めない区分。stable/beta は同時 2 版までパス版で提供し、非推奨表明から最低 12 か月は維持する。 | stability tier, 安定性区分 |
| Deprecation | stable/beta の interface を将来削除する予告。deprecated_since 以降は応答に Deprecation ヘッダを付与し、sunset_at が定まれば Sunset ヘッダも付与する。sunset_at は deprecated_since から最低 12 か月後でなければならない。 | 非推奨化 |

## Standards

### Web Content Accessibility Guidelines 2.2

W3C Recommendation — https://www.w3.org/TR/WCAG22/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WCAG22-KEYBOARD | required | MUST | すべての認証操作をキーボードだけで完了可能にする。 |
| WCAG22-FOCUS | required | MUST | フォーカスを視認可能にし重要な要素が完全に隠れないようにする。 |
| WCAG22-LABELS-ERRORS | required | MUST | 入力にラベルを付け、エラーをテキストで識別して修正方法を示す。 |
| WCAG22-STATUS | required | MUST | 認証結果や送信エラーをフォーカス移動なしに支援技術へ通知する。 |

### General Data Protection Regulation

Regulation (EU) 2016/679 — https://eur-lex.europa.eu/eli/reg/2016/679/oj

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| GDPR-CONSENT-WITHDRAWAL | required | MUST | ResourceOwner が同意を撤回でき、撤回後の新規発行へ利用しない。Consent / ConsentLifecycle は OAuth2 context が所有する。 |
| GDPR-ERASURE | required | MUST | 削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う。 |
| GDPR-PROCESSING-RECORDS | required | MUST | セキュリティ・認可イベントの監査記録を定義済み期間保持する。保持期間は Audit context が所有する。 |
