---
status: pending
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
