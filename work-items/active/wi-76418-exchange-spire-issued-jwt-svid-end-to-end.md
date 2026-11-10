---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: [wi-77559-verify-jwt-svid-against-spiffe-rules, wi-88550-read-spiffe-bundle-endpoint-format]
change_kind: maintenance
spec_impact: { kind: none, reason: "REQ-WORKLOADIDENTITY-001 から 007 と REQ-OAUTH2-089 の規範を変えず、実際の SPIRE が発行した JWT-SVID でその規範が成り立つことを確かめるテストだけを足す。" }
---

# 実際の SPIRE が発行した JWT-SVID で、Token Exchange までを確かめる

## 動機

アテステーションの交換のテストは、テスト内で作った鍵と JWT を使う。
`spiffe://` 形式の値は出てくるが、SPIRE が発行した JWT-SVID と、SPIRE の Bundle Endpoint が返すバンドルでの確認はない。
[wi-77559](../done/wi-77559-verify-jwt-svid-against-spiffe-rules.md) と [wi-88550](../done/wi-88550-read-spiffe-bundle-endpoint-format.md) が足す規則は、実物の形（`iss` の有無、`kid`、鍵の `use`、`spiffe_refresh_hint`）に依存するため、テスト用に作った値だけでは解釈の誤りを検出できない。

## 対象範囲

- SPIRE のサーバーとエージェントをコンテナで起動し、JWT-SVID を発行させるテストを足す。
- そのバンドルを信頼設定に登録し、Token Exchange で IdMagic のアクセストークンを得られることを確かめる。
- 別のトラストドメインの SPIRE が発行した JWT-SVID が拒否されることを確かめる。
- `aud` を発行者識別子にした SPIRE の JWT-SVID をクライアントアサーション（`spiffe_jwt`、REQ-OAUTH2-089）として送り、`client_credentials` のトークンを得られることを確かめる。wi-25266 が実装した経路で、SPIRE が `jti` を付けない JWT-SVID を繰り返し使えることも含む。
- Docker が要るテストとして、既存の Docker に依存するテスト（`15b65565` の OpenLDAP の分離）と同じ実行区分に置き、`mise` タスクから実行できるようにする。

## 対象外

- 本番の SPIRE の運用手順。
- X.509-SVID。

## 設計

SPIRE の起動は、OpenLDAP のテストと同じく Docker に依存するテストの区分に置き、通常の `test-go` からは外す。
SPIRE のイメージのバージョンは固定し、依存の更新の棚卸しの対象にする。

採らない案：SPIRE の発行物をファイルとして保存し、テストで再生する。
有効期限が短く、保存した JWT-SVID はすぐ期限が切れ、時刻を偽るとテストの意味が薄れるため採らない。

## タスク

- [ ] T001 [Test] SPIRE を起動し、JWT-SVID とバンドルを得るテストの部品を作る。
- [ ] T002 [Test] 通常経路と、別のトラストドメインの拒否を確かめるテストを足す。
- [ ] T003 [Tooling] `mise` タスクと CI の実行区分を足す。
- [ ] T004 [Verify] 検証を実行する。

## 検証

- 追加する `mise` タスク（Docker を使うため、サンドボックスの外で実行する）。
- `mise run check-work-items`。

## リスク

- SPIRE の起動が遅く、CI の時間が延びる。Docker に依存するテストの区分に置いて通常の検証から外す。
