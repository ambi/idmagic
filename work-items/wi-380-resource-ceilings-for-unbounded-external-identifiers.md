---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-16
change_kind: bugfix
priority: p2
affected_spec:
  - { path: spec/contexts/saml/models.tsp, symbol: Product.Saml.SamlServiceProvider }
  - { path: spec/contexts/ws-federation/models.tsp, symbol: Product.WsFederation.WsFedRelyingParty }
---

# 外部が値を決める識別子に資源上限としての天井を与える

## Motivation
`wi-128-string-length-limits-policy` は、上限を持つ文字列の単位と適用点を揃えた。一方、外部が値を決める識別子（`entity_id`、`wtrealm`、`scim_id`、`kid` など）には「外部仕様が長さを定めていない値を DB 都合で短く切らない」という理由で上限を置かなかった。

しかし「上限なし」もまた、検討された選択ではなく既定値でしかない。これらは連携相手が値を決めるため、こちらの都合で短くはできないが、無制限に受けてよい理由もない。次の 2 点が具体的な問題である。

- **すでに壊れている**。`entity_id`、`wtrealm`、`scim_id` は主キーの構成要素であり、`kid` は主キーそのものである。PostgreSQL の btree v4 は索引行 1 件を 2704 バイトに制限するので、これを超える値は挿入時点で失敗する。実測（`postgres:17`、非圧縮データ）では次のとおり。

  | 長さ | 結果 |
  |---|---|
  | 2000 バイト | 成功 |
  | 2704 バイト | `ERROR: index row size 2720 exceeds btree version 4 maximum 2704 for index "..._pkey" (SQLSTATE 54000)` |
  | 8192 バイト | `ERROR: index row requires 8208 bytes, maximum size is 8191 (SQLSTATE 54000)` |

  つまり上限は事実上すでに存在し、ただしそれが宣言されておらず、超過は 500 として低水準のエラー文言で返る。`wi-128` が長さ違反に与えた「422 とフィールド名つきの `detail`」という扱いから漏れている。値が圧縮しやすいかどうかで閾値が動くため、再現しにくい形でもある。
- **標準が上限を定めている値がある**。少なくとも SAML 2.0 の entity identifier は「1024 文字以下の URI」と規定されている（`saml-core-2.0-os` の Entity Identifier）。`wi-128` はこれを「外部仕様で長さが明確でない識別子」に分類したが、これは誤りだった。実装前に原典で確認する。

## Scope
- 外部が値を決める識別子ごとに、上限の根拠を「標準が定めている」「資源上限として置く」のどちらかに分類する。対象は少なくとも `saml_sp_trusts.entity_id`、`saml_*.entity_id`、`wsfed_rp_trusts.wtrealm`、`scim_user_refs.scim_id`、`scim_group_refs.scim_id`、`signing_keys.kid`。
- 標準が定める値は原典を引いて記録する。定めていない値には、btree の索引行上限より内側で、実運用の実例を拒否しない天井を置く。
- `spec/SPECIFICATION.md` の String length limits に、資源上限の区分とその根拠を追記する。「上限を置かない値もある」という現在の記述を改める。
- Go の domain / usecase で天井を強制し、`wi-128` が用意した `spec.LengthError` 経由で 422 として返す。DB には `CHECK` を最後の防壁として置く。
- 主キー成分ではない外部由来の文字列（token hash、`tls_client_auth_subject_dn`、`quarantine_reason` など）についても、天井を置くか置かないかを判断して記録する。

## Out of Scope
- 235 個ある `TEXT` 列すべてへ機械的に上限を設定すること。
- 既存データを切り詰めること。棚卸しで違反行が見つかった場合は、天井を広げるか移行手順を用意する。
- `JSONB` の payload、配列要素数、ネスト深さ。これらは文字列長ではなく `HTTP_MAX_BODY_BYTES` と別の手段が扱う。

## Plan
- 先に既存データを棚卸しし、列ごとの現存最大長を報告する。`CHECK` の追加はその後に行う。連携先が実際に使っている識別子を拒否しないことが、この work item の成否を分ける。
- 天井は btree の 2704 バイトより内側に置く。ただし単位はコードポイントなので、マルチバイトの識別子でもバイト上限に達しないよう、コードポイント数の天井はその 1/3 を目安にする。

## Tasks
- [ ] T001 [Inventory] 対象列と現存最大長、および外部標準が定める上限の有無を調査して報告する。
- [ ] T002 [Spec] 資源上限の区分と根拠を String length limits に追記する。
- [ ] T003 [Domain] 天井を Go 側で強制し、422 として返ることを確認する。
- [ ] T004 [Postgres] `CHECK` を追加し、psqldef が収束することを確認する。
- [ ] T005 [Tests] 標準が許す実例を拒否しないこと、天井 ±1、btree 上限に達する前に 422 で止まることを検証する。

## Verification
- `just check-spec`
- `just check-schema`
- `just verify`
- 手動確認: 各識別子について、上限の根拠が標準の引用か資源上限の説明として記録されている。

## Risk Notes
外部プロトコル識別子に短すぎる上限を置くと相互運用性を壊す。実装は、標準に明記のある値（SAML entity identifier）から先に進め、標準が沈黙している値は実データの棚卸しを経てから決める。天井は「安全のための資源上限」であって「業務上望ましい長さ」ではないので、迷ったら広く取る。
