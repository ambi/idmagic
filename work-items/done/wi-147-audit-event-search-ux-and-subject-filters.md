---
depends_on: [wi-146-extract-audit-bounded-context]
status: completed
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
OAuth2 系イベントはユーザー名検索に一致しない。

レビュー時に、当初案 (トップレベルの「よく使う条件」+ 検索属性一覧という二層 UI、および username を
tenant salt 付き hash で相関する設計) 自体が分かりにくさ・複雑さの原因になっていると判断し、以下の
方針に転換した。

## Scope
- **decisions**:
  - [[ADR-046]] の username 条項 (tenant salt 付き hash を first-class とする方針) を撤回し、
    [[ADR-104]] で置き換える。username は平文で扱い、hash 化・7 日後の redaction sweep はしない。
  - 同時に、IP / User-Agent / device fingerprint の hash 化・truncate も撤回する (ADR-104 の対象を
    拡張。位置情報の country-code-only 方針は変更しない)。
- **scl**:
  - `spec/contexts/audit.yaml`: `AuditEventQuery` / `AuditEventSearchAttribute` /
    `ListAdminAuditEvents` / `ExportAdminAuditEvents` の説明を、「誰を検索するか」の軸が
    検索属性一覧に一本化される語彙へ更新する。`AuditEventQuery.username` (検索時に
    user_id へ解決する) を追加する。`event.type` / `outcome` を allowlist / enum 由来の
    選択肢として扱う方針を追記し、`GetAdminAuditEventSearchOptions` インターフェースを追加する。
  - `spec/contexts/authentication.yaml` / `spec/contexts/oauth2.yaml`: `usernameHash` /
    `ipHash` / `ipTruncated` / `uaHash` / `deviceFingerprintHash` を全廃し、`username` (失敗系
    イベントのみ) / `ip` / `userAgent` / `deviceFingerprint` の平文フィールドに置き換える。
    7 日 redaction sweep の invariant を削除する。
- **go**:
  - 実アカウントが常に確定するイベント (`UserAuthenticated`、`ConsentGranted` などの OAuth2
    フロー系イベント) は username を payload に持たない。管理 UI が username で検索する場合は
    監査 HTTP ハンドラが `UserRepo.FindByUsername` で `user_id` に解決してから、既存の高速な
    `user_id` フィルタで検索する (該当ユーザーが存在しない場合は 0 件)。
  - 実アカウントが確定しない可能性があるイベント (`AuthenticationFailed`) は、平文 username を
    そのまま監査検索の `actor.username` 属性として使う (`raw_storable: true` / `transform: none`)。
  - IP / User-Agent / device fingerprint も同様に平文へ統一する。
    `backend/authentication/usecases/retention.go` の `FailureUsernamePlaintextDays` /
    `AuthenticationFailureUsernameRedactor` を削除する。
  - `event.type` / `outcome` の選択肢を返す `GetAdminAuditEventSearchOptions`
    (`GET /api/admin/audit_events/search_options`) を追加し、`auditEventCategoryTypes` を
    単一の正として UI に提供する。
- **ui**:
  - `ui/src/features/admin-audit-events/AdminAuditEventsPage.tsx` を、「誰を検索するか」を含む
    すべての検索条件を1つの「検索属性」一覧に統一する形へ全面整理する。トップレベルに残すのは
    開始日時・終了日時・最大件数のみとし、イベントカテゴリも検索属性の一覧内 (`event.type` 行の
    グループ選択肢) に統合する。
  - 検索属性の選択肢: 「ユーザー ID (操作者)」「ユーザー名 (操作者、実在アカウントを検索時に解決)」
    「ログイン試行のユーザー名 (失敗記録、実在しないアカウントも含む)」「対象ユーザー (ユーザー ID)」
    「イベント種別 (カテゴリの一括選択を含む)」「結果」「IP アドレス」「セッション ID」。
    どのイベントにも実装されていない `transaction.id` は選択肢から削除する。
  - 検索条件を URL query string と同期し、初期表示・再読み込み・共有 URL で同じ検索結果を復元する。
  - 検索属性行の select / input / remove button の高さと配置を揃える。
- **tests**:
  - URL query からの初期化、検索実行時の URL 更新、共有 URL での loader 検索、
    username → user_id 解決 (成功 / 該当なし)、AuthenticationFailed の平文 username 検索を検証する。

## Out of Scope
- 監査イベントストアの保持期間そのもの (日数の見直し)、削除、エクスポート形式、SIEM 連携。
- 監査 bounded context への切り出し。これは [[wi-146-extract-audit-bounded-context]] の範囲。
- 新しい監査イベント種別の追加。
- 任意 SQL / JSONPath / OData / SCIM filter の公開。検索は既存の registry allowlist に閉じる。
- `LoginThrottled` / `AuthenticationEventAggregated` の `keyHash` (rate-limit bucket key、監査検索
  registry には出ていない別用途) は対象外。位置情報の country-code-only 方針も変更しない。

## Plan
- ADR-104 で ADR-046 の username / IP / User-Agent / device fingerprint 条項を撤回し、平文へ統一する。
  位置情報 (country code only) は変更しない。
- UI は「誰を検索するか」を含むすべての検索軸を1つの一覧に統一し、トップレベルには日付・件数だけを残す。
  category は event.type 行のグループ選択肢として統合する。
- `event.type` の選択肢は Go の `auditEventCategoryTypes` を単一の正とし、新設エンドポイントで UI に提供する。
- URL 同期は `/admin/audit_events?...` を正とし、loader が query string を API に渡して初期結果を取得する。
- OAuth2 フロー系イベントのユーザー名検索は、emit 時 payload に何も足さず、検索時に
  `UserRepo.FindByUsername` で `user_id` に解決してから既存の `user_id` フィルタで検索する。
  `AuthenticationFailed` のみ平文 username をそのまま検索属性として使う。

## Tasks
- [x] T001 [ADR] ADR-104 を作成し ADR-046 の username / IP / UA / device fingerprint 条項を撤回する。
- [x] T002 [SCL] `authentication.yaml` / `oauth2.yaml` の `usernameHash` / `ipHash` / `ipTruncated` /
  `uaHash` / `deviceFingerprintHash` を平文フィールドへ置き換え、7 日 redaction sweep の invariant を削除する。
- [x] T003 [SCL] `audit.yaml` に `AuditEventQuery.username` と `GetAdminAuditEventSearchOptions` を追加し、
  検索属性一覧への統合を反映した語彙に更新する。
- [x] T004 [Go] `events.go` / `consent.go` / `authentication_event_attributes.go` /
  `authorize_handler.go` / `retention.go` / 永続化層から hash 関連コードを除去し、平文フィールドへ統一する。
- [x] T005 [Go] 監査 HTTP ハンドラに `UserRepo` を配線し、`username` クエリパラメータを
  `user_id` へ解決するロジック (該当なしは 0 件) と `GetAdminAuditEventSearchOptions` を実装する。
- [x] T006 [UI] 検索フォームを「誰を検索するか」を含む単一の検索属性一覧へ再構成し、
  トップレベルを日付・件数のみにする。`transaction.id` を選択肢から削除する。
- [x] T007 [UI] 検索条件を router URL と同期し、共有 URL / reload / browser navigation で同じ検索結果を復元する。
- [x] T008 [Test] Go / UI / e2e で、検索意味、username 解決 (成功 / 該当なし)、
  AuthenticationFailed の平文検索、URL 同期を検証する。
- [x] T009 [Verify] `just yaml-check`、`just scl-render`、`just verify-go`、`just verify-ui`、
  必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
  - reason: 管理 UI の検索フォームと URL 共有は browser behavior を含むため、主要シナリオは e2e で確認する。
- 手動確認:
  - `/admin/audit_events` の検索属性一覧だけで、ユーザー ID・ユーザー名・イベントカテゴリ・イベント種別を
    含むすべての条件を組み立てられる。
  - 存在しない username で検索すると 0 件になる (エラーにも全件にもならない)。
  - `UserAuthenticated` と OAuth2 フロー系イベントが、同じユーザー名検索で横断的に見つかる。
  - `AuthenticationFailed` は実在しないユーザー名でも検索できる。

## Risk Notes
監査検索は調査・コンプライアンスに使われるため、UI の語彙が曖昧だと誤った調査結果を導く。ADR-046 の
username / IP / UA / device fingerprint 条項撤回は、既存の retention sweep・永続化層・関連テストにも
影響するため、変更範囲を漏れなく洗い出してから着手する。username 未解決時に「フィルタ無視で全件返す」
という誤動作を作らないよう、0 件応答を明示的にテストする。

## Completion

- **Completed At**: 2026-07-14
- **Summary**:
  ADR-104 で ADR-046 の username / IP / User-Agent / device fingerprint 条項を撤回し、平文のまま
  扱う方針に転換した (位置情報の country-code-only 方針は変更なし)。`UserAuthenticated` と OAuth2
  フロー系イベント (`ConsentGranted` / `AuthorizationCodeIssued` / `AuthorizationCodeRedeemed` /
  `AccessTokenIssued` / `RefreshTokenIssued`) は username を payload に持たず、管理 UI の
  `AuditEventQuery.username` が検索時に `UserRepo.FindByUsername` で `user_id` へ解決してから
  既存の `user_id` 経路で絞り込む (該当なしは 0 件)。`AuthenticationFailed` のみ、実在しない
  アカウント名も追跡する必要があるため平文 username をそのまま `actor.username` 検索属性として使う。
  `backend/authentication/usecases/retention.go` の 7 日 redaction sweep 機構と対応する Postgres
  query (`RedactAuthenticationFailureUsernames`) を削除した。
  UI (`AdminAuditEventsPage.tsx`) は「誰を検索するか」を含むすべての検索条件を1つの検索条件一覧へ
  統合し、トップレベルには開始日時・終了日時・最大件数だけを残した。イベントカテゴリも検索条件一覧内の
  1行 (`quick.category`) として扱う。`transaction.id` (どのイベントにも実装されておらず常に0件になる
  死んだ選択肢) を削除した。検索条件は `validateSearch`/`loaderDeps` で URL query string と同期する。
  実装中のレビューで、当初案 (トップレベル + 検索属性一覧の二層 UI、username の tenant-salt hash 相関)
  は複雑さの原因と判断され上記の設計へ転換した。
- **Affected Guarantees State**:
  `actor.username` / `client.ip` を含む監査検索属性は平文で保存・検索される (hash 化・truncate はしない)。
  実アカウントが常に確定するイベントは username を payload に持たず、検索時解決のみで相関する。
  `AuditEventQuery.username` は該当ユーザーが存在しない場合に必ず 0 件を返し、フィルタ無視で全件返す
  誤動作にはならない。監査検索 UI の「誰を検索するか」は1つの検索条件一覧に一本化されている。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just verify-go` — passed (golangci-lint 0 issues、全 backend パッケージのテスト green)
  - `just verify-ui` — passed (format/lint/typecheck/build + vitest all green)
  - e2e (`ui-scenario-actions.spec.ts` の audit log シナリオ) — passed
  - 実運用環境相当の手動検証で、以下 3 件のバグを発見し修正・再検証済み:
    1. `shared/adapters/http/server/routes.go` が監査 HTTP 層への配線で非推奨のテスト互換フィールド
       `d.UserRepo` (実運用の bootstrap では常に nil) を参照しており、username 検索が常に 0 件になっていた。
       `d.IdentityManagement.UserRepo` に修正。
    2. 不正な `user_id` (Postgres の UUID 列にキャストできない値) を検索するとページ全体がクラッシュして
       いた。原因は2つ: (a) pgx v5 の `Query()` は遅延実行のため型キャストエラーが `rows.Err()` 側で
       顕在化しており、`Query()` の戻り値だけを見ていた修正では効かなかった。(b)
       `backend/audit/adapters/persistence/postgres/` に他の全 context に存在する `harness_test.go`
       (`TestMain` で embedded-postgres を起動する配線) が無く、監査コンテキストの Postgres 統合
       テストが一度も実際の DB に対して実行されていなかった (wi-146 切り出し時の抜け)。両方を修正し、
       `harness_test.go` を追加した上で実際の embedded-postgres に対して green を確認した。
    3. ルートローダーが検索条件由来の取得失敗をそのまま投げており、ページ全体が汎用の認証エラー画面に
       落ちていた。ページ内のエラー表示 (既存の `Alert` state) に留めるよう修正した。
- **Evidence**:
  - 実行日: 2026-07-14
  - 実行環境: ローカル開発環境 (macOS)、embedded-postgres (docker 不使用)
  - 実行主体: Claude (Sonnet 5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
