---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "共有ライブラリがテナント ID を受け取る経路と、テナントを解決する middleware の置き場所だけを変える。テナントの解決規則、テナントの分離、HTTP の応答、永続状態は変えない。" }
---

# 共有ライブラリが Tenancy に依存せずにテナント ID を受け取るようにする

## 動機

`tools/check/boundary-debt.json` の `shared-dependency` のうち 5 件は、共有ライブラリから Tenancy への依存である。

| 共有ライブラリ | 使っているもの |
| --- | --- |
| `http/support_http` | テナントを解決して文脈へ載せる middleware |
| `ratelimit/db_postgres`、`security/salts_memory`、`security/salts_postgres` | 文脈からのテナント ID の取得（`tenantports.TenantID`） |
| `storage/db_memory` | テナントの型 |

保存先と塩の共有ライブラリが必要とするのはテナント ID の文字列だけである。
共有ライブラリが Tenancy に依存すると、共有ライブラリを経由してほかのモジュールが Tenancy へ到達し、共有ライブラリの変更が Tenancy の型の変更に引きずられる。

## 対象範囲

- 保存先と塩の共有ライブラリが、テナント ID を Tenancy の型なしで受け取るようにする。
- テナントを解決する middleware を Tenancy へ移し、組み立て地点で結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- テナントの解決規則と、テナントのない文脈を拒否する規則の変更。

## 設計

### 今回の負債整理での見送りと再開条件

2026-10-18 時点で残るのは、support_http、rate limiter、二つの salt の保存先、メモリ保存先の計5件である。
署名器からの Tenancy 依存は wi-91505 で解消した。
middleware だけを移しても、support_http.Deps.TenantRepo と Cookie の名前や属性を決める処理に Tenancy の型への依存が残る。
メモリ保存先の DefaultTenant と TenantKey も、空の ID を default へ補う意味を持ち、型名を文字列へ写すだけでは共有の条件を満たさない。

今回は HTTP と背景処理の文脈の契約を混ぜる変更を見送る。
wi-73872 と合わせて、解決済みの設定を持つ文脈と ID だけの文脈、未解決の文脈の区別を先に決める。
再開条件は、テナント ID の読取りと未解決時の拒否を一か所に保ち、設定を持たない背景処理に空の Tenant を解決済みとして見せず、HTTP の origin と Cookie の分離も保つ設計をこの項目へ記録することである。
単に共有の getter を増やして、二つの文脈キーの優先順位と default の規則を別々に維持する実装は行わない。

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：テナントの解決規則の所有者は Tenancy なので、middleware は Tenancy に置く。
- D5：テナント ID の文字列を受け取るだけなら、共有ライブラリの条件を満たす。

| 案 | 判断の材料 |
| --- | --- |
| 共有ライブラリの操作がテナント ID を引数で受け取る | 呼び出し側の変更が最も多いが、依存は最も単純になる |
| テナント ID だけを運ぶ文脈の受け渡しを共有ライブラリに置き、Tenancy が値を載せる | 呼び出し側は変わらない。[テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)が退けたのは Tenancy の型を運ぶ共有化であり、文字列だけなら D5 を満たすかを確かめる。テナントのない文脈を default へ落とさない規則（`REQ-TENANCY-006`）を二か所で保たないことが条件になる |

### 再開時の設計

利用者の承認により、wi-73872 の ID だけの文脈への移行も同じブランチで行う。
入力は `REQ-TENANCY-006`、`REQ-TENANCY-007`、`REQ-TENANCY-009`、解決済み文脈の読み手、既存のメモリ保存先の空 ID の補完である。
D1 により解決、Cookie の属性の判断と default の補完は Tenancy が所有し、D5 により共有側には文字列の ID と HTTP の位置情報の運搬だけを置く。
D4 により共有側が最小の `Scope` インターフェースを定義し、Tenancy の非公開の解決済み文脈が実装する。

| 案 | 判断 |
| --- | --- |
| 全操作に ID 引数を増やす | 保存ポートと全呼び出し元の変更が必要で、HTTP の位置情報と設定の区別は別途必要になるため採らない |
| ID と解決済み設定を別の文脈キーで運ぶ | 上書き時に異なるテナントの設定が残り、二つの ID の優先順位が必要になるため採らない |
| 一つの文脈キーへ ID を公開する値を格納する | 採用。解決済み値だけが Tenancy の設定と HTTP の位置情報を持つ。ID だけの値へ上書きすると古い設定と位置情報も見えなくなる |

`shared/tenantcontext` に `Scope interface { TenantID() string }`、`WithScope(context.Context, Scope) context.Context`、`WithID(context.Context, string) context.Context`、`Current(context.Context) Scope`、`TenantID(context.Context) string` を置く。
ID の読取りと空 ID の拒否は `TenantID` 一か所だけに置き、Tenancy の既存 getter はそこへ委譲する。
HTTP の位置情報は任意の `HTTPLocation() (issuer, prefix string, hostOnlyCookies bool)` から読み、ID だけの文脈には存在しない。
Tenancy の `WithTenantID` は ID だけの値を格納し、`Tenant` は解決済みの非公開型だけを返す。
停止中のテナントの Job の判定や保存処理は変えず、解決のための Repository 読取りも追加しない。

解決 middleware は `tenancy/internal/handlers_http` へ移し、Tenancy の公開組み立て操作から server_http が結ぶ。
`support_http.Deps.TenantRepo` は各利用側の依存へ移す。
メモリ保存先は、共有の `DefaultScope` と `ScopedKey` に補完値を明示的に渡す。
default を選ぶ互換規則は各保存先の所有者が適用し、共有側は任意の補完値を受け取る文字列の操作だけを行う。
Tenancy のポートへ補完関数を移す案は、Saml と WsFederation から Tenancy への新しい循環依存を作ることが境界検査で分かったため採らない。
共有のページング関数はそのまま残す。
テナントの設定が増える変更は Tenancy の値の中だけ、HTTP の経路変更は Tenancy の middleware と組み立ての中だけに閉じる。
二重の getter を保つ仮定や新しい業務上の規則は残さない。

主要な検査は既存の解決、origin、Cookie、保存先の契約と `REQ-TENANCY-006` の未解決文脈の拒否である。
新しい ID だけの値と、解決済み値から上書きした際の設定の消去を先にテストする。
変更行のミューテーションの比較基準は `60b18f90d` とする。

## 計画

文脈のテストを先に追加し、共有の受け渡し、HTTP の所有者、メモリ互換処理の順に移す。
その後、ID だけの本番呼び出し元を置き換え、保存先の契約、組み立てを含むテスト、境界検査を行う。

## タスク

- [x] T001 [Design] 案を比較して受け取り方を決める。
- [x] T002 [App] 共有ライブラリと呼び出し側、middleware を書き換える。`test-go-package` で文脈、メモリ共通処理、HTTP 共通処理と Tenancy の HTTP 境界を検査する。`TestTenantIDOnlyContext` は部分的な Tenant を返す旧表現を拒否し、`TestTenantIDOnlyContextReplacesResolvedSettings` と `TestTenantIDOnlyContextDropsCookieScope` は古い設定の引継ぎを検査する。未解決の拒否は `REQ-TENANCY-006` を参照する既存と追加のテストで固定する。
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [x] T004 [Verify] 変更を検証する。Cookie 属性の公開は `TestResolvedTenantPublishesHTTPLocation` で供給側からも検査する。HTTP 配線は `TestRealmDiscoveryUsesTenantIssuer` が host と path の取り違えを検出する。規範と HTTP 契約は変更せず、追加の受け入れ RED は不要とし、ID だけの文脈の局所 RED と既存の組み立てテストを使う。

## 検証

- `mise run report-contract-evidence -- --changed --base-revision 60b18f90d` で共有コードの影響先を照合する。影響する未解析操作に参照の欠落はない。
- `mise run test-go-package` の対象は `./backend/tenancy/ports`、`./backend/shared/tenantcontext`、`./backend/shared/storage/db_memory`、`./backend/shared/http/support_http`、`./backend/tenancy/internal/handlers_http` とする。ID の拒否、HTTP の属性、所有者を移した処理の既存契約を検査する。
- `mise run test-go-mutation` は同じ五つの変更ロジックのパッケージについて、第5引数に `60b18f90d` を渡し、変更行だけを選ぶ。残るパッケージは import、呼出し先または明示引数だけの変更であり、新たな条件式を持たない。
- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify` で共有コードの全逆依存、正式な HTTP 配線、各メモリ保存先の共有契約、race、仕様とツールの整合を検査する。公開 HTTP 契約と UI の操作は変えていないため、ブラウザーの追加操作は選ばない。

## リスク

- テナント ID の受け渡しを変えるときに、テナントのない文脈が default テナントへ落ちると、テナントの分離が破れる。
  `REQ-TENANCY-006` のテストが新しい経路を通ることを確かめる。

## 要約

共有ライブラリから Tenancy への依存5件を解消し、負債台帳から削除した。
テナントの ID、設定と HTTP の位置情報を一つの格納先で運び、ID の読取りと未解決時の拒否を共有側の一か所へ集めた。
解決 middleware と Cookie 属性の判断は Tenancy が所有し、メモリ保存先が使う補完値は各所有者が明示する。
承認された wi-73872 も併せて実装し、背景処理の13箇所とテストの ID だけの文脈を移行した。
`spec-diff` は規範差分なしを示した。解決の規則、HTTP 応答、永続状態とイベントは変えていない。
変更行の変異には意味のある生存を残さず、文字列の加算を減算に置き換える8件はビルド不能の変異として分類した。
テナントの解決規則の変更と、背景処理で設定を追加解決する変更は対象外とした。
