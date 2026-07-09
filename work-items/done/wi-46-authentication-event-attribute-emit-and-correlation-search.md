---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-21
---

# 認証イベントに username / IP を PII-safe な検索属性として emit し、相関検索 UI を載せる

## Motivation
[[wi-44-authentication-event-store-and-search]] は `UserAuthenticated` /
`AuthenticationFailed` に産業標準の属性 (usernameHash / ipTruncated / ipHash /
uaHash / countryCode / deviceFingerprintHash / riskScore) を後方互換で
**フィールドとして用意した**が、実際の emit 経路 (ブラウザログイン / 失敗記録) はまだ
これらに値を設定しておらず、ADR-046 の hash / truncation も実装されていない。

汎用イベントログ検索基盤 (検索属性 registry / filter 式 / sidecar ストア / tenant salt store /
相関ハッシュ統一) は [[wi-145-generic-audit-event-search-foundation]] で先に用意する。本 WI は
その基盤の上に、**最初の PII-safe 検索属性として username / IP を載せる**。emit 経路で
usernameHash / ipTruncated / ipHash / uaHash を tenant salt 付きで抽出し、admin が平文で入力した
ユーザー名 / IP をサーバ側で hash / 丸めに変換して相関検索できるようにする。これにより
credential stuffing の相関調査を実現する。ADR-046 の PII 列を新規に扱うため、本 WI の差分は
レビュアー 2 名以上で確認する。

## Dependencies
- [[wi-145-generic-audit-event-search-foundation]] (registry / filter / sidecar / salt store /
  相関ハッシュ統一)。本 WI は 145 完了後に着手する。

## Scope
- **decision**:
  - 平文 username は失敗イベント限定で 7 日保持し sweep で null 化する (ADR-045 / ADR-046)。
    成功イベントは sub / actor.id を持つため平文を保管しない。
  - username (lowercased) / IP の SHA-256、IPv4 /24・IPv6 /48 への丸めを wi-145 の単一 correlation
    helper (`SaltedHash` / `TruncateIP`) に集約したまま、emit 経路と検索変換で共有する。
- **scl**:
  - wi-145 で追加した `AuditEventSearchAttribute` registry に `actor.username` / `client.ip` を
    PII-safe (hash / ip_truncate、tenant salt 要) 属性として UI 表示可能で宣言する。
    `UserAuthenticated` / `AuthenticationFailed` の PII 属性 (usernameHash / ipTruncated / ipHash /
    uaHash) が emit 値を持つことを仕様に反映する。
- **go**:
  - emit ヘルパ: ブラウザログイン成功 (`UserAuthenticated`) / 失敗 (`AuthenticationFailed`) で
    actor / client / outcome と usernameHash / ipTruncated / ipHash / uaHash を抽出する。IP は
    `extractClientIP` 由来を /24・/48 で丸め、hash は tenant salt 付き SHA-256。失敗イベントは
    平文 username も 7 日分だけ載せる。
  - `ExtractSearchAttributes` に `actor.username` / `client.ip` の PII 属性抽出を接続する。
  - 失敗イベント平文 username の 7 日 null 化を retention sweep (ADR-045) に追加する。
- **ui**:
  - 監査ログ検索を検索ビルダーに拡張する。初期プリセットとして「ユーザー名」「IP アドレス」
    「イベント種別」「結果」「対象ユーザー」「セッション / トランザクション」を選べるようにし、
    username / IP は平文入力をサーバ側で hash / 丸めに変換して検索する。

## Out of Scope
- 汎用検索基盤 (registry / filter / sidecar / salt store / 相関ハッシュ統一)。
  [[wi-145-generic-audit-event-search-foundation]] で実装済み前提。
- GeoIP 連携による country_code 解決 (当面 "" のまま。別 WI)。
- device fingerprint の収集と deviceFingerprintHash の算出。
- risk_score の算出 (フィールド確保のみ)。
- impersonation 機能本体と本人通知 (wi-44 / ADR-041 の通り別 WI)。

## Plan
- SCL-first で `actor.username` / `client.ip` を UI 表示可能な検索属性として明示し、
  `UserAuthenticated` / `AuthenticationFailed` の PII-safe emit 値と失敗時 username retention の
  保証を descriptions / scenarios / invariants に反映する。
- Go は既存の wi-145 helper (`SaltedHash` / `TruncateIP` / `NormalizeUsername`) を単一の正として使い、
  emit 経路で authentication event の payload に hash / truncation を載せ、sidecar 抽出器は payload の
  transform 済み値だけを読む。平文 username は失敗イベント payload にのみ残す。
- `client.ip` の sidecar 値は registry の transform と検索変換に合わせて `ipTruncated` を使う。
  `ipHash` は payload に記録するが、本 WI の検索軸は `client.ip` = truncated network とする。
- retention は既存 sweep の delete policy と分離し、7 日を超えた失敗イベント payload の
  `username` を null 化する repository 境界を追加する。未対応 store では no-op にせず、memory /
  PostgreSQL の両方で実装する。
- UI は既存の監査ログページに filter builder を足す。URL query は wi-145 の
  `filter=field:op:value[,value2]` を直接使い、既存 `type` / `category` / `user_id` は残す。

## Tasks
- [x] T001 [SCL] `actor.username` / `client.ip` の UI 表示と authentication event PII-safe emit、
  失敗 username 7 日 null 化を SCL に反映する。`just yaml-check`。
- [x] T002 [Go/usecases] authentication event enrichment helper と `ExtractSearchAttributes` の
  PII 属性抽出を追加し、hash / truncation / tenant 分離のテストを追加する。`just test-go`。
- [x] T003 [Go/adapters] browser login 成功 / 失敗 emit 経路で usernameHash / ipTruncated /
  ipHash / uaHash を設定し、server-side filter 変換で検索一致する handler テストを追加する。
  `just test-go`。
- [x] T004 [Go/retention] 失敗イベント payload.username の 7 日 null 化を memory / PostgreSQL に
  実装し、retention sweep へ配線する。`just test-go && just lint-go && just build-go`。
- [x] T005 [UI] 監査ログページに検索ビルダーを追加し、username / IP / event.type / outcome /
  target.id / session.id / transaction.id を指定できるようにする。`just verify-ui`。
- [x] T006 [Verify] `just verify`、completion 追記、`done/` へ移動、commit。

## Verification
- `just test-go`
  - reason: search attribute 抽出、hash / 丸めの単一ヘルパの正しさ、tenant salt の分離
    (同一ユーザー名でも tenant が違えば hash が異なる)、平文入力 → サーバ hash / 丸め化での検索一致、
    失敗イベント平文 username の 7 日 null 化。
- `just lint-go` / `just build-go`
- `just typecheck-ui` / `just lint-ui` / `just build-ui`
- 手動: 誤パスワードでログイン失敗 → 監査ログ検索ビルダーで `actor.username` に同じユーザー名
  (平文) を入力すると当該失敗イベントが相関ヒットする。`client.ip` でも同様に絞り込める。
  イベント種別・結果・期間フィルタと併用できる。

## Risk Notes
hash / truncation を誤ると個人情報が監査ログに平文で流れるため (ADR-046)、属性抽出と検索変換は
wi-145 の同一の単一ヘルパを使う。PII 検索属性を増やす本変更はレビュアー 2 名以上で確認する。
tenant salt の取り違えは cross-tenant 相関の漏洩につながるため、salt の取得は必ず対象 tenant に
紐づけて行う。

## Completion

- **Completed At**: 2026-07-10
- **Summary**:
  認証イベントの username / IP / user-agent を PII-safe な監査検索属性として emit するようにした。
  `UserAuthenticated` / `AuthenticationFailed` payload に `usernameHash` / `ipTruncated` /
  `ipHash` / `uaHash` を追加し、sidecar 検索属性 `actor.username` / `client.ip` は transform 済み値だけを
  保存する。admin 監査ログ UI には検索属性ビルダーを追加し、username / IP / event.type / outcome /
  target.id / session.id / transaction.id を AND 条件で検索できるようにした。
  - username は `NormalizeUsername` 後に tenant salt 付き `SaltedHash` で保存・検索する。
  - client IP は検索用に `TruncateIP` の IPv4 /24・IPv6 /48 値を保存し、payload には salt 付き
    `ipHash` も保持する。
  - `AuthenticationFailed.username` の平文は retention sweep で 7 日超過後に JSON null 化し、
    `usernameHash` は相関検索用に保持する。
  - 既存の `user_id` query param と UI の sub フィルタ名を揃え、従来のカテゴリ / 期間 / export は互換維持。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-go` - passed
  - `just lint-go` - passed
  - `just build-go` - passed
  - `just verify-ui` - passed
  - `just verify` - passed
- **Affected Guarantees State**:
  `AuditPIISearchAttributesAreTransformed` と `AuthenticationFailurePlainUsernameTtl` を SCL に追加し、
  実装・テスト済み。sidecar は username / IP の平文を保存せず、tenant salt による cross-tenant
  username hash 分離、IP 丸め検索、失敗イベント username の 7 日 null 化を確認した。
