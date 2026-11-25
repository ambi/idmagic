---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-26
priority: p2
group: development-process
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "上流のトークンエンドポイントに対する前提を、記録の再生と状態を持つ模倣で確かめる検査を足し、記録の再生と模倣の照合をモジュールに依らないテスト支援へ移す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。検査が規範に反する実装を見つけたら、その時点で affected_spec を宣言し直す。" }
---

# 上流の OAuth 2.0 トークンエンドポイントへの前提を、記録の再生と模倣で確かめる

## 動機

[wi-422](../done/wi-422-external-contract-and-failure-verification.md) は、消費する側の SCIM の契約を、RFC の例を転記した記録の再生と、記録を忠実さの基準にした状態を持つ模倣で確かめるようにした。
往復 7 件の転記と模倣一つの費用で、PATCH の 204 とフィルターの値の符号化という二つの誤りが見つかった。
wi-422 の棚卸しは Provisioning、Sourcing、SharedSignals の三つに限っていたが、外向きの HTTP を送る本番コードはほかにもある（2026-11-26 に確かめた）。

上流のトークンエンドポイントは二つのモジュールが呼ぶ。
Provisioning は下流の SCIM への認証にクライアント資格情報フロー（RFC 6749 §4.4）を使い、TokenVault は委任したトークンの取得と更新に上流のトークンエンドポイントを使う。
どちらのテストも、要求ごとに一つの応答を返す `httptest` の関数であり、相手の実際の応答の形を固定していない。
トークンの更新が壊れると、委任したトークンが失効し、下流への同期が止まる。

記録の再生と模倣の照合は、wi-422 で Provisioning の内部のテスト支援に置いたので、ほかのモジュールから使えない。
この項目は二つのモジュールにまたがる最初の利用側なので、共有する置き場所もここで決める。
上流の OIDC IdP、Vault と OpenBao の Transit、上流の MCP サーバーの項目は、この項目が決めた置き場所を使う。

## 対象範囲

- 記録の形式（取得日、相手の版、取得方法を必須とする）、再生器、模倣と記録の照合を、モジュールに依らないテスト支援へ移す。SCIM の記録と模倣は Provisioning に残し、共有の部品を使うように直す。
- RFC 6749 §4.1.3、§4.4、§5.1、§5.2、§6 の要求と応答の例を転記した記録と、Provisioning のトークン取得と TokenVault の取得と更新の再生の検査。
- 状態を持つトークンエンドポイントの模倣。発行、更新、更新トークンの回転、`invalid_grant`、`invalid_client`、`429`、`5xx`、不正な本文、`expires_in` の欠落を返せる。記録の再生と同じ応答を返すことを検査する。
- 見つかった誤りの修正。製品の結果を変えるものは、着手後に利用者に確かめてから `affected_spec` を宣言する。

## 対象外

- 上流の OIDC IdP の Discovery、ID トークン、JWKS。[wi-65625](wi-65625-verify-upstream-oidc-identity-providers-with-recordings.md) が扱う。
- IdMagic が提供するトークンエンドポイントの適合試験。[wi-33](wi-33-protocol-conformance-ci.md) が扱う。
- 実在の上流への接続。協力も試験用の契約も得られず、CI で再現できない。

## 設計

共有の置き場所は、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)に従って着手時に比べ、記録をこの節に書く。
候補は、`backend/shared/testing` の下のテスト支援、相手の種類ごとの複製、wi-422 の Provisioning の内部に置いたままにする案である。
テスト支援は本番の入口から import せず、そのことを `mise run check-boundaries` で確かめる。

記録の扱いは [テスト方針](../../docs/development/testing.md#外部契約と設定の利用) に従う。

## 計画

1. 共有の置き場所を比べ、記録の形式、再生器、照合を移す。SCIM の検査が移動の前後で同じ結果になることを確かめる。
2. RFC 6749 の例を転記し、Provisioning と TokenVault の再生の検査を置く。
3. トークンエンドポイントの模倣を置き、記録との照合と、記録にない組み合わせの検査を置く。

## タスク

- [ ] T001 [Design] 共有の置き場所を比べて記録し、部品を移す。
- [ ] T002 [Adapter] RFC 6749 の記録と再生の検査を置く。
- [ ] T003 [Adapter] トークンエンドポイントの模倣と照合を置く。
- [ ] T004 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 記録の応答を RFC 6749 に反する形へ変えると、再生の検査が落ちる。
- 模倣の応答を記録から外すと、照合が食い違いを報告する。
- `mise run check-boundaries` と `mise run verify-changed -- main`。

## リスク

部品を移す間に SCIM の検査の照合が緩むと、wi-422 の保証が気付かれずに弱まる。
移動の前後で、模倣を記録から外す否定の検査が同じ食い違いを報告することを確かめる。
