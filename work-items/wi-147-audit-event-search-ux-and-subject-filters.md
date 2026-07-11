---
depends_on: [wi-146-extract-audit-bounded-context]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# 監査イベント検索の条件 UI とユーザー相関検索を、管理者が意味を誤解しない形に整理する

## Motivation
監査イベント検索は `category` / `user_id` / `filter` による絞り込みを提供しているが、管理 UI では
「イベントカテゴリ」と検索属性の「イベント種別」、「対象ユーザー (sub)」と検索属性の「対象ユーザー」が
並んで表示され、どれが何を検索するのか判断しづらい。さらに `sub` は OIDC の用語であり、現在の UI では
ユーザー ID を指しているにもかかわらず、管理者向けラベルとして露出している。

検索属性にも同じ問題がある。「ユーザー名」と「対象ユーザー」の違いが明確でなく、対象ユーザーが ID なのか
名前なのか分からない。イベント種別は SCL で定義されたイベント型の allowlist であるべきだが、UI は自由入力に
しているため、存在しない値を入力できる。結果やセッション、トランザクションも、入力すべき値の由来が UI 上で
分からない。

機能面では、検索実行後も URL が変わらないため、監査調査の条件を URL として保存・共有できない。また、
`UserAuthenticated` はユーザー名で検索できる一方、`ConsentGranted` / `AuthorizationCodeIssued` /
`AccessTokenIssued` / `AuthorizationCodeRedeemed` / `RefreshTokenIssued` など、ユーザーが特定される
OAuth2 系イベントはユーザー名検索に一致しない。ユーザー ID を「対象ユーザー (sub)」に入れた場合だけ一致する
ため、同じ対象ユーザーの行動を横断して追うという監査イベント検索の価値が弱い。

## Scope
- **scl** (`spec/contexts/audit.yaml`。[[wi-146-extract-audit-bounded-context]] 完了により
  監査 API / models の所有 context は audit へ移設済み):
  - `AuditEventQuery` / `AuditEventSearchAttribute` / `ListAdminAuditEvents` /
    `ExportAdminAuditEvents` の説明を、トップレベル条件と検索属性 filter の役割が重複しない語彙へ更新する。
  - `sub` / `user_id` / `target.id` / `actor.username` の意味を明確化し、管理 UI に露出する語彙を
    「ユーザー ID」「ユーザー名」「操作者」「対象ユーザー」などへ整理する。
  - `event.type` と `outcome` を allowlist / enum 由来の選択肢として扱う方針を追記する。
- **go**:
  - 監査検索属性 extractor が、ユーザーを特定できるイベントで一貫して `actor.username` または
    `target.username` 相当の検索属性を出せるかを見直す。
  - `ConsentGranted` / `AuthorizationCodeIssued` / `AccessTokenIssued` /
    `AuthorizationCodeRedeemed` / `RefreshTokenIssued` など OAuth2 フロー系イベントで、ユーザー名・ユーザー ID
    の相関検索が同じ意味で効くようにする。
  - `event.type` / `outcome` の選択肢を UI が機械的に参照できる形で公開するか、既存生成物から安全に
    導出する。
- **ui**:
  - `ui/src/features/admin-audit-events/AdminAuditEventsPage.tsx` の検索フォームを整理する。
  - 検索条件を URL query string と同期し、初期表示・再読み込み・共有 URL で同じ検索結果を復元する。
  - 検索属性行の select / input / remove button の高さと配置を揃える。
  - `event.type` と `outcome` は自由入力ではなく選択式にする。
  - セッション ID / トランザクション ID は、詳細 payload からコピーして検索できる内部相関 ID であることが
    ラベルや placeholder から分かるようにする。UI に長い説明文は増やしすぎず、必要なら短い補助文・tooltip を使う。
- **tests**:
  - URL query からの初期化、検索実行時の URL 更新、共有 URL での loader 検索、OAuth2 系イベントの
    ユーザー名検索を検証する。

## Out of Scope
- 監査イベントストアの保持期間、削除、エクスポート形式、SIEM 連携。
- 監査 bounded context への切り出し。これは [[wi-146-extract-audit-bounded-context]] の範囲。
- 新しい監査イベント種別の追加。
- 監査イベント payload 全体の再設計や過去データの大規模 backfill。
- 任意 SQL / JSONPath / OData / SCIM filter の公開。検索は既存の registry allowlist に閉じる。

## Plan
- SCL-first で検索条件の意味を整理する。特に `AuditEventQuery.user_id` と filter の `target.id` が
  同じ対象を指すのか、片方を後方互換用として UI から隠すのかを決める。
- UI は「よく使う条件」と「詳細検索属性」を分ける。イベントカテゴリは分類、イベント種別は具体的な
  event type、ユーザー名は人間が入力するログイン名、ユーザー ID は payload / user detail で確認できる
  stable identifier として表記する。
- `event.type` の選択肢は SCL events または Go registry 由来に寄せ、UI 側の手書きリストを最小化する。
  すぐに機械生成できない場合でも、少なくとも `AuditSearchRegistry` / SCL と drift しない検証を置く。
- URL 同期は `/admin/audit_events?...` を正とし、loader が query string を API に渡して初期結果を取得する。
  フォーム送信は router navigation で URL を更新し、API 呼び出しと表示状態を URL から再構築できるようにする。
- OAuth2 フロー系イベントのユーザー名検索は、emit 時 payload に username を足すのか、既存 user ID から
  extractor 側で解決するのかを実装前に決める。PII governance と既存の tenant salt / hash 方針を崩さない。

## Tasks
- [ ] T001 [SCL] `AuditEventQuery` と `AuditEventSearchAttribute` の語彙・説明を更新し、トップレベル条件と filter の使い分け、ユーザー ID / ユーザー名 / actor / target の意味を明文化する。
- [ ] T002 [SCL] `event.type` / `outcome` を選択式として扱う仕様を追記し、UI が参照する選択肢の正を決める。
- [ ] T003 [Go] OAuth2 系イベントの検索属性抽出を見直し、ユーザーが特定されたイベントはユーザー名・ユーザー ID で一貫して検索できるようにする。
- [ ] T004 [Go/API] URL query string と API query の変換を後方互換込みで検証し、必要ならイベント種別・結果の選択肢取得方法を追加する。
- [ ] T005 [UI] 監査イベント検索フォームのラベル、項目分割、placeholder、control 高さを整理する。
- [ ] T006 [UI] 検索条件を router URL と同期し、共有 URL / reload / browser navigation で同じ検索結果を復元する。
- [ ] T007 [Test] Go / UI / e2e で、検索意味、OAuth2 系イベントのユーザー検索、URL 同期、選択式フィールドを検証する。
- [ ] T008 [Verify] `just yaml-check`、`just scl-render`、`just verify-go`、`just verify-ui`、必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
  - reason: 管理 UI の検索フォームと URL 共有は browser behavior を含むため、主要シナリオは e2e で確認する。
- 手動確認:
  - `/admin/audit_events` でイベントカテゴリ・イベント種別・ユーザー名・ユーザー ID の違いが UI 上で分かる。
  - `event.type` / `outcome` は存在する選択肢から選ぶ。
  - セッション / トランザクションの検索値が詳細 payload から辿れる。
  - 検索実行後の URL を別タブで開くと同じ検索結果になる。
  - `UserAuthenticated` と OAuth2 フロー系イベントが、同じユーザー名検索で横断的に見つかる。

## Risk Notes
監査検索は調査・コンプライアンスに使われるため、UI の語彙が曖昧だと誤った調査結果を導く。`sub` はプロトコル
用語としては正しいが、管理者 UI ではユーザー ID として明示する。ユーザー名検索の対象イベントを広げる際は、
PII を平文 sidecar に保存しない既存方針を維持し、tenant salt / hash 変換の境界を崩さない。URL に検索条件を
載せるため、機密値を検索条件として扱う場合の露出リスクもレビューする。
