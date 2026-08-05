---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-025: Device Authorization Grant (RFC 8628) の実装

## コンテキスト

`spec/flows/device-code-flow.json` に device flow の状態機械、`spec/grants/grant-types.json`
に device_code グラント、`spec/discovery.json` に `device_authorization_endpoint` が
宣言済みだった。しかし HTTP / usecase 実装が無く、**Discovery が広告する
`/device_authorization` が 404 を返す**状態だった。入力制約のあるデバイス（TV・CLI・IoT）は
このグラントでしか認可を受けられないため、適合性と実用性の両面で欠落していた。

RA の観点では、これは「仕様核（状態機械・grant matrix・discovery）が既に存在するので、
アダプタと usecase は仕様から再生成できる」ことを実証する好機でもある。

## 決定

（ADR-001 の device-code 状態機械を実装に落とす）

`POST /device_authorization` で device_code / user_code を発行し、`/device` で
verification_uri（user_code の入力・承認・拒否）を提供し、`/token` に
`urn:ietf:params:oauth:grant-type:device_code` のポーリング分岐を追加する。`device_code` は
32 バイト乱数でベアラ秘密のため SHA-256 ハッシュのみ保存し、`user_code` は母音・紛らわしい
文字を除いた 20 文字集合 × 8 桁で `WDJB-MJHT` 形式に表示する。状態遷移は
`spec/flows/device-code-flow.json` の遷移テーブルに従い、approve/deny/exchange を勝手に
実装しない。ポーリングは `authorization_pending` / `slow_down` / `access_denied` /
`expired_token` を返し、interval と slow_down 増分は仕様核を権威とする。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 影響

- `DeviceCodeStore` ポート + memory/valkey アダプタ、`device-routes.ts`、
  3 usecase（request / verify / exchange）、`device-authorization.ts` ドメインを追加。
- `token-routes.ts` に device_code 分岐、`OAuthErrorCode` に
  `authorization_pending` / `slow_down` / `expired_token` を追加。
- 仕様核には新規ファイルを足さず、既存の状態機械・grant matrix・discovery を「実装が追いつく」
  形で消費した（RA の再生成可能性の実証）。

## セキュリティ上の注意

- user_code は低エントロピーなので、verification_uri での入力レート制限が production では必須
  （本アプリは枠組みのみ）。`DeviceAuthorizationDenied` の多発を SIEM で総当たり検知できる
  よう監査イベントを設計した。
- device_code はハッシュ保存。`slow_down` でポーリング濫用を抑制。

## 却下した代替案

- **device_code をそのまま保存**: ベアラ秘密の平文保存はアンチパターン。ハッシュのみ保存。
- **user_code に英数字フル集合**: 視認混同（0/O, 1/I）と偶発的な単語生成を避けるため子音集合に限定。
- **承認画面を 2 ステップ (enter_user_code → approve) に分割**: 本アプリの UX を単純化し、
  1 フォームで enter_user_code + approve を連続適用する（状態機械上は両遷移を順に発火）。
