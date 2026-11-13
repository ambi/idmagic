---
status: cancelled
---

# テナントの停止、再開、正規ロケーション切替を対話セッションに限定する

2026-11-14 に中止し、範囲を [[wi-468-system-console-privileged-session-assurance]] へ統合した。
テナントの制御面の経路は、実行時すでに API アクセストークンを受けていない（制御面の `Authenticator` に API トークンの照合を配線していない）。
残るのは `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` の `x-api-token-scopes` が `tenants:write` のままである宣言の不一致だけである。
wi-468 は制御面の変更操作に MFA のステップアップを要求し、その前提として対話セッションを必要とするので、宣言の修正も同じ項目で行う。
