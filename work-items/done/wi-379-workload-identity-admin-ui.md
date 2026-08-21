---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-16
priority: p2
depends_on: [wi-54-workload-identity-federation-spiffe]
change_kind: feature
initial_context:
  specification:
    - spec/contexts/workloadidentity/SPECIFICATION.md#REQ-WORKLOADIDENTITY-008
    - spec/contexts/workloadidentity/SPECIFICATION.md#REQ-WORKLOADIDENTITY-009
  typespec:
    - IdMagic.WorkloadIdentity.Operations.ListWorkloadTrustBundles
    - IdMagic.WorkloadIdentity.Operations.RefreshWorkloadTrustBundleJWKS
    - IdMagic.WorkloadIdentity.Operations.CreateAgentWorkloadBinding
    - IdMagic.Contract.WorkloadTrustBundleResponse
    - IdMagic.Contract.AgentWorkloadBindingResponse
    - IdMagic.Contract.WorkloadAttestationRejected
  source:
    - backend/workloadidentity/handlers_http/routes.go
    - frontend/src/features/admin-mcp-resource-servers
    - frontend/src/features/admin-agents
    - frontend/src/api/admin.ts
    - frontend/src/lib/adminNav.ts
  tests:
    - frontend/src/features/admin-mcp-resource-servers
  stop_before_reading:
    - backend/workloadidentity/verification_jose
    - backend/workloadidentity/db_postgres
affected_spec:
  - { path: spec/contexts/workloadidentity/scenarios.md, requirement: REQ-WORKLOADIDENTITY-008 }
---

# ワークロード ID の信頼設定とバインディングを管理コンソールから運用できるようにする

## Motivation

`WorkloadIdentity` は 13 本の管理 API を持ち、`WorkloadTrustBundle` の登録・更新・無効化・再有効化・削除・JWKS 再取得と、`AgentWorkloadBinding` の作成・無効化・再有効化・削除を提供する (REQ-WORKLOADIDENTITY-008)。しかし `frontend/src` に `workload` は **0 件**である。同じ管理対象である `admin-agents` と `admin-mcp-resource-servers` には画面があるのに、ワークロード ID だけが API 専用のまま残っている。

信頼バンドルは、外部の発行者が署名した JWT を IdMagic のトークンへ交換してよいと宣言する設定である。これは「長期シークレットを配布せずにエージェントへ資格情報を渡す」という設計の要であり、同時に**登録を誤れば任意の発行者がエージェントになりうる**設定でもある。`subject_pattern` はグロブで、曖昧な一致は拒否されるとはいえ、どのパターンがどのエージェントへ向いているかを一覧できなければ、意図しない広さの設定に気付けない。

運用事故を想定すると欠落がはっきりする。上流の鍵が漏れた疑いがあるとき、運用者はまず「どの信頼バンドルが有効か」を見て、当該のものを無効化し、JWKS を再取得する必要がある。現在それを行う手段は API を直接叩くことだけである。認証情報の配布経路を止める操作が、画面から行えないまま残っている。

`WorkloadTrustBundleJWKSRefreshed` や `WorkloadAttestationRejected` (11 種の理由コードを持つ) といったイベントも既にあるが、参照する場所が無い。アテステーションが拒否されている原因を運用者が特定できない。

## Scope

- 管理コンソールに `WorkloadIdentity` の機能を追加する。既存の `admin-agents` / `admin-mcp-resource-servers` と同じ構成・同じ命名規則に従う。
- 信頼バンドルの一覧・詳細・登録・更新・無効化・再有効化・削除・JWKS 再取得を画面から行えるようにする。一覧では `trust_domain` / `issuer` / 状態 / `jwks_cached_at` が読めること。
- バインディングの一覧・作成・無効化・再有効化・削除を、所属する信頼バンドルの詳細から行えるようにする。`subject_pattern` と対象 `Agent` の対応が一覧で読めること。
- 直近の `WorkloadAttestationRejected` を理由コードとともに提示し、設定の誤りを運用者が特定できるようにする。理由コードは英語の識別子のままにせず、UI 文言として `ja` / `en` の両方を辞書へ置く。
- 破壊的操作 (削除、無効化) には確認を挟む。削除はカスケードでバインディングを消すため、影響範囲を確認画面に示す。
- 既存の権限をそのまま使う。`workload-identity:read` / `workload-identity:write` は配線済みであり、新しい権限は作らない。

## Out of Scope

- 仕様と Go 実装の変更。API は 13 本すべて存在し、本 work item は画面を足すだけである。表示に必要な項目が API に無いことが判明した場合のみ、その差分を Scope へ繰り入れる。
- `WorkloadIdentity` の HTTP エラー応答の RFC 9457 移行。[[wi-338-workloadidentity-context-problem-details-migration]] が持つ。
- X.509-SVID と SPIRE サーバーの同梱。[[wi-54-workload-identity-federation-spiffe]] が将来送りと判断済みで、その判断は変えない。
- 複数の資格情報バインディングを持つ `Agent` に対する選択規則。現行実装は最初のバインディングを採るが、これは仕様に無い挙動であり、画面で見えるようにはするが規則の設計は行わない。

## Design

### 画面構成

`admin-agents` の親子構成 (一覧 → 詳細 → 編集) を踏襲し、`admin-mcp-resource-servers` の平坦な構成は採らない。ルートは 4 本である。

| ルート | 画面 | `PageMarker` の種別 |
|---|---|---|
| `/admin/workload-identity` | 信頼バンドルの一覧。`trust_domain` / `issuer` / 状態 / `jwks_cached_at` を読める | `admin-workload-identity` |
| `/admin/workload-identity/new` | 信頼バンドルの登録 | `admin-workload-trust-bundle-create` |
| `/admin/workload-identity/{id}` | 信頼バンドルの詳細。バインディングとアテステーション拒否を同居させる | `admin-workload-trust-bundle-detail` |
| `/admin/workload-identity/{id}/edit` | 信頼バンドルの更新 | `admin-workload-trust-bundle-edit` |

バインディングに専用ルートを与えず、詳細画面へ埋め込む。バインディングは所属する信頼バンドルの外では意味を持たず (`subject_pattern` の一意性も交換時の曖昧判定も、同一 `trust_bundle_id` の中でしか成立しない)、単独で開ける画面は「どの発行者に対するパターンか」を落とした状態を作ってしまう。REQ-WORKLOADIDENTITY-005 が拒否する曖昧な一致は同じバンドル内の複数パターンが原因であり、それを見つけられる場所は一覧が並ぶ画面だけである。作成もインラインの 1 行フォームとし、専用ルートを作らない。

登録と更新は逆に専用ルートへ出す。`issuer` と `trust_domain` は不変で、変更するには登録し直すしかない (`WorkloadTrustBundleUpdateRequest` がこの 2 つを持たない)。この非対称は入力欄を無効化して示す必要があり、一覧に畳めるフォームには収まらない。

### アテステーション拒否の提示

専用の読み取り経路は作らず、既存の監査検索を再利用する。`NewEmitFunc` がすべての `DomainEvent` を `AuditEventRepo` へ写しており、`WorkloadAttestationRejected` は `tenantId` を持つのでテナントで絞り込める。管理 API の `type` クエリは許可リストを通さない完全一致であるため、`GET /api/admin/v1/audit_events?type=WorkloadAttestationRejected` がそのまま使える。

`trustBundleId` は「issuer が登録済みだった場合のみ設定する」ため、詳細画面ではこれで絞り、一覧ではテナント全体の直近を出す。`unregistered_issuer` は定義上 `trustBundleId` を持たず、どのバンドルにも属さないまま一覧側にだけ現れる。これは欠落ではなく、未登録の発行者が叩いている事実そのものである。

理由コードは 11 種すべてを辞書に置く。`reason` は `spec/contexts/workloadidentity/models.tsp` の `WorkloadAttestationRejected` が正本で、そこに列挙された値と辞書の鍵を 1 対 1 で対応させる。未知の値は辞書を引けないので生の識別子へ落とす (握り潰すと、仕様が増えたときに画面から消える)。

### 破壊的操作

削除と無効化は確認ダイアログを挟む。信頼バンドルの削除はバインディングをカスケードで消すため、確認文にそのバンドルが現在持つバインディング件数を埋め込む。件数は詳細画面が既に読み込んでいるものを使い、確認のために追加の取得を行わない。

### 権限による出し分け

`workload-identity:read` のみで変更操作を隠す、という当初の想定は実現できない。この 2 つのスコープは API アクセストークン専用であり、管理コンソールはブラウザセッション (`/api/auth/account` が返すのはロールだけ) で動く。ロールは `admin` か否かの二値で、読み取り専用の管理者という状態が存在しない。したがって画面側に出し分ける根拠が無い。

ゲートは既存のロール境界 (`requirePortalAccount('admin')` とサーバ側の `RequireAdmin`) のままとし、新しい権限も読み取り専用モードも作らない。読み取り専用の管理者を UI に持ち込むなら、それはブラウザセッションに権限を載せる変更であって、本 work item の範囲ではない。T005 はこの事実の記録と、提示ロジックの単体テストに置き換える。

### 監査検索を借りることの限界

監査の検索軸は `backend/audit/ports/audit_search_attribute.go` の閉じた許可リストで決まり、そこに `trustBundleId` は無い。したがって信頼バンドル単位の絞り込みは取得後に画面側で行うしかなく、他のバンドルの拒否が取得窓を埋めると、拒否が存在するのに詳細画面が「該当なし」に見えうる。

軸そのものを増やすのは Go と仕様の変更であり、本 work item の Out of Scope である。代わりに、窓を使い切ったことを `truncated` として画面へ渡し、空表示の文言を「無い」から「この範囲には無い」へ切り替える。黙って誤った答えを出す代わりに、答えの範囲を明示する。軸の追加は [[wi-377-agent-and-delegation-chain-audit-axes]] が既存の許可リストへ手を入れる際にまとめて扱うのが筋である。

### 資格情報バインディングが複数ある対応先

交換は `Agent` が持つ最初の `AgentCredentialBinding` を採るが、この選択規則は仕様に無い。規則の設計は Out of Scope のままとし、バインディング一覧で対応先の `client_ids` が 2 つ以上ある行に注意書きを出す。「どれが使われるか定まらない」という状態そのものは、規則を決めなくても見せられる。

### ルートツリーの再生成

`just generate-routes` を新設した。`routeTree.gen.ts` は Vite のプラグインが dev と build で生成するが、`bun run build` は Vite を起動する前に型検査を走らせる。新しいルートファイルを足した直後は、それを宣言するツリーが書かれる前に型検査が落ちるため、既存の recipe だけでは進めない。生成だけを単体で回す口を用意してこの循環を切る。

## Plan

- 既存の管理機能の構成をそのまま写す。フロントエンドの表示ロジック分離の方針 (提示ロジックを分離し、単体テスト可能にする) に従う。
- UI 文言は `*.i18n.ts` の辞書に `ja` / `en` の両方を置く。テストは辞書の値を参照し、翻訳済み文字列を直接書かない。
- アテステーション拒否の提示は、既存の監査検索を再利用できるかを最初に確認する。専用の読み取り経路を作るのは、再利用できないと分かってからにする。

## Tasks

- [x] T001 [Design] 信頼バンドルとバインディングの画面構成を確定し `## Design` に記録する。
- [x] T002 [Adapters] 13 本の管理 API に対応する API クライアントと型を追加する (`frontend/src/api/admin.ts`、`frontend/src/types.ts`)。
- [x] T003 [UI] 信頼バンドルの一覧・詳細・作成・更新・状態変更・削除・JWKS 再取得を実装する。RED → GREEN: `AdminWorkloadTrustBundlesPage` 7 件、`AdminWorkloadTrustBundleFormPages` 5 件 (REQ-WORKLOADIDENTITY-008)。
- [x] T004 [UI] バインディングの一覧・作成・状態変更・削除を、詳細画面に埋め込んで実装する。RED → GREEN: `AdminWorkloadTrustBundleDetailPage` 10 件 (REQ-WORKLOADIDENTITY-009)。
- [x] T005 [UI] アテステーション拒否の提示と、理由コードの `ja` / `en` 辞書を追加する。RED → GREEN: `presentation > rejectionReasonLabel` が 11 種すべてを両ロケールで引く。
- [x] T006 [Verify] 提示ロジックの単体テストを書き (`presentation.test.ts` 34 件)、権限による出し分けが成立しない理由を `## Design` に記録した。

## Verification

- `just verify-ui`
- `just verify`
- 手動: 信頼バンドルを登録し、バインディングを作成し、K8s の ServiceAccount トークンでトークン交換が通ること、バンドルを無効化すると交換が拒否され、その拒否が理由コードとともに画面に現れることを確認する。

## Risk Notes

リスクは low。既存 API に画面を足すだけで、仕様も認可も変えない。

ただし信頼バンドルの削除はバインディングをカスケードで消すため、確認画面で影響範囲を示さないと、一覧から軽い気持ちで消せてしまう。破壊的操作の確認は実装時に省略しない。

## Completion
- **Completed At**: 2026-08-16
- **Summary**:
  ワークロード ID を管理コンソールから運用できるようにした。13 本の管理 API はすべて存在していたが `frontend/src` に `workload` が 1 件も無く、上流の鍵が漏れた疑いがあるときに信頼バンドルを無効化して JWKS を取り直す手段が API を直接叩くことしか無かった。その経路を画面に出した。
  画面は一覧・登録・詳細・編集の 4 ルートで、バインディングには専用ルートを与えず詳細に埋め込んだ。バインディングは所属する信頼バンドルの外では意味を持たず (`subject_pattern` の一意性も曖昧な一致の判定も同一 `trust_bundle_id` の中でしか成立しない)、単独で開ける画面は「どの発行者に対するパターンか」を落とした状態を作るためである。登録と更新だけは逆に専用ルートへ出した。`issuer` と `trust_domain` は更新リクエストが受け付けず、変更するには登録し直すしかないという非対称を、入力欄の無効化で示す必要があるからである。
  アテステーション拒否の提示に専用の読み取り経路は作らなかった。すべての `DomainEvent` は共通の配信点から `audit_events` へ写され、`WorkloadAttestationRejected` は `tenantId` を持つので、既存の監査検索を `type` 完全一致で引くだけで足りる。11 種の理由コードは仕様の `WorkloadAttestationRejected.reason` を正本として `ja` / `en` の辞書へ 1 対 1 で写し、辞書に無い値は生の識別子へ落とす (握り潰すと、仕様が理由を増やしたときに画面から消える)。`trustBundleId` は issuer が登録済みだった場合にのみ載るため、`unregistered_issuer` はどの詳細画面にも現れず一覧にだけ現れる。これは欠落ではなく、未登録の発行者が叩いている事実そのものなので、一覧側で「どの信頼バンドルにも属さない」と明示した。
  破壊的操作は確認を挟み、確認文と実際に走る操作を 1 つの値に束ねて保持するので、確認したものと実行するものがずれる経路が無い。信頼バンドルの削除の確認文には、そのバンドルが現在持つバインディングの件数を埋める。件数は詳細画面が既に読み込んでいるものを使い、確認のための追加の取得はしない。
  当初の想定にあった「`workload-identity:read` のみの権限で変更操作を隠す」は実現できないと判明したので、実装せず理由を `## Design` に記録した。この 2 つのスコープは API アクセストークン専用で、管理コンソールはロールしか持たないブラウザセッションで動く。読み取り専用の管理者という状態が存在しない以上、画面側に出し分ける根拠が無い。
- **Semantic Difference** (`just spec-diff`):
  - `no normative specification change against main`。仕様は変えていない。本 work item は既存の 13 本の管理 API (REQ-WORKLOADIDENTITY-008 / -009) に画面を足すだけで、表示に必要な項目はすべて既存の応答に揃っていた。Out of Scope の「表示に必要な項目が API に無い場合のみ差分を Scope へ繰り入れる」は発動しなかった。
- **Verification Results**:
  - `just verify` - passed (check / check-api-compat / test-tools / typecheck-tools / lint-go / test-go / format-check-ui / lint-ui / test-ui-unit / build-ui)
  - `just verify-ui` - passed
  - `just spec-diff` - 正規仕様の差分なし
  - UI 単体テスト 63 件を追加 (提示ロジック 37、一覧 7、詳細 14、フォーム 5)。
  - `/code-review` を Standards と Spec の 2 軸で実施し、指摘のうち次を修正した。
    - 更新時に `jwks_uri` を常に送っていた不具合 (RED → GREEN)。Go 側は非 nil なら空文字でも上書きするため、インライン JWKS だけのバンドルで名前を直しただけの更新が取得元設定を空にしていた。変更が無ければ送らず、意図して空にした場合だけ空文字を送る (それが唯一の消し方である) ように改めた。
    - 拒否の取得窓を使い切った場合に「直近の拒否はありません」と断言していた点。窓の範囲を明示する文言へ切り替えた。
    - `frontend/scripts/generate-routes.ts` のコメントが英語だった点 (`tools/**` 以外の TypeScript コメントは日本語)。
    - `agentDisplayName` と信頼バンドル名の解決が同じ形で二重化していた点を `displayNameForID` に寄せ、`TrustBundleStatusBadge` をバインディングにも使う実態に合わせて `EnabledStatusBadge` へ改名。詳細画面が 1 つの実体を `trustBundle` と `bundle` の 2 つの名前で持っていた点も解消。
    - 辞書の `audience` の表記を、既存の用法 (`Audience`) へ揃えた。
  - 手動確認は未実施。K8s の ServiceAccount トークンによる実際のトークン交換を要するため、実クラスタでの確認は残っている。
- **Follow-ups**:
  - `just generate-routes` を新設した。`bun run build` が Vite より先に型検査を走らせるため、新しいルートファイルを足した直後は `routeTree.gen.ts` が書かれる前に型検査が落ちる。生成だけを単体で回す口でこの循環を切った。
  - 複数の資格情報バインディングを持つ `Agent` に対する選択規則は設計していない。現行実装が最初のバインディングを採る挙動は仕様に無いままである (Out of Scope のとおり)。画面には注意書きを出すところまで行った。
  - 監査に `trustBundleId` の検索軸が無いため、詳細画面の拒否一覧はテナント全体の直近 50 件を取ってから絞り込む。範囲を使い切ったことは画面に出るが、絞り込み自体はサーバ側へ寄せるべきである。[[wi-377-agent-and-delegation-chain-audit-axes]] が許可リストへ手を入れる際にまとめて扱う。
  - 破壊的操作の確認ダイアログは `admin-consents` にほぼ同一のものがある。共通コンポーネントへ引き上げるのが筋だが、既存機能の変更を伴うため本 work item では行わなかった。
  - `WorkloadIdentity` の HTTP エラー応答は RFC 9457 未移行のまま。[[wi-338-workloadidentity-context-problem-details-migration]] が持つ。
