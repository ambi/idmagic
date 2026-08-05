---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-041: 認証イベントを通常イベントと bucket 集約の 2 系統で持つ

## コンテキスト

production IdP は「誰が・いつ・どの手段で・どこから認証したか / 失敗したか」を時系列で
保持し、admin が調査できる必要がある (Keycloak の login events、Okta の System Log 相当)。
一方で credential stuffing / brute force 時には失敗イベントが秒間数千件のオーダーで発生し、
1 失敗 = 1 行で素朴に INSERT すると次の 2 つが同時に壊れる:

1. **可用性**: 監査ストア (Postgres) への書き込みが攻撃トラフィックで詰まり、正規の認証
   経路まで巻き添えで遅くなる。
2. **調査性**: 数百万行の同一パターン失敗に埋もれ、admin が異常を読み取れず MTTR が悪化する。

[[ADR-029]] の throttle は試行**そのもの**を遅らせる防御だが、ロック後も試行は届き続ける。
そのため throttle とは別に、**監査イベント側でも爆発を吸収する**仕組みが要る。

## 決定

認証イベントを 2 系統で持つ: 通常イベント (`authentication_events`、1 アクション = 1 行) と
bucket 集約 (`authentication_event_buckets`、`(tenantId, kind, keyHash, 5 分窓)` 単位で畳み込んだ
計数)。[[ADR-029]] の throttle lockout 閾値に到達したアクター由来の失敗は、以後 individual な
`AuthenticationFailed` を emit せず bucket へ切り替える — throttle と bucket は同じ tenant-salt
付き `keyHash` を共有し、平文は監査に流さない。1 窓につき `AuthenticationEventAggregated` を
1 件だけ emit し、以後の増分はイベントを増やさず bucket 行の `count` を更新する。既定の集約閾値
(per-account 10 / per-IP 50 / per-tenant 1000、窓 5 分) は per-account を ADR-029 の account
lockout 閾値と揃え、正規ユーザの打ち間違いを個別に観察可能なまま残す。impersonation イベントは
本人保護のため bucket 集約と retention 短縮の対象外とする。既存 `UserAuthenticated` /
`AuthenticationFailed` の payload 拡張は破壊的変更を避けるため全て optional とする。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Authentication event logging セクションにある。

## 影響

- bucket モードは in-memory counter + 5 分単位 flush で動き、攻撃時も個別 INSERT を出さない。
  この挙動は in-memory ([[wi-20-authentication-event-history]]) と Postgres 永続 ([[wi-44-authentication-event-store-and-search]])
  の双方で同一に保つ。
- 認証成功・失敗・MFA の結果は共通の `AuthenticationOutcomeBus` を通し、bucket 切替判定を
  bus 層へ集約する (個々の use case が切替ロジックを持たない)。
- 通常イベントと bucket は別テーブルだが、admin 検索 UI では同一タイムラインに混在表示し、
  bucket 行はドリルダウン可能な集約行として描く。

## 却下した代替案

- **1 失敗 = 1 行のみ (bucket なし)**: 実装は単純だが攻撃時にストレージと調査性が破綻する。
- **サンプリング (N 件に 1 件だけ記録)**: 計数の正確さが失われ、攻撃規模の評価ができない。
  bucket は全件を `count` に積むため規模を保持する。
- **throttle ロック後はイベントを一切残さない**: 攻撃の発生事実と規模が監査から消えるため不可。
  bucket により「1 行 + count」で痕跡と規模の両方を残す。
