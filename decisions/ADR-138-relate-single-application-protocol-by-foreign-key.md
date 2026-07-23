---
status: accepted
authors: [tn]
created_at: 2026-07-24
---

# ADR-138: Application と単一 protocol 設定を nullable 外部キーで関連付ける

## コンテキスト

Application catalog と OAuth2 client、SAML service provider、WS-Fed relying party の関係は、
Application の JSON 配列に protocol 種別と opaque key を保存していた。この形は外部キー、一意性、
cascade、join をデータベースで表現できず、参照解決のために tenant の Application 全件と JSON を
走査する必要があった。

当初は複数 protocol を束ねられる柔軟性を残していたが、実際の作成・編集・利用フローでは一つの
Application が複数 protocol を持つユースケースはない。一方、protocol 設定には Dynamic Client
Registration や trust 管理 API から作られ、catalog に掲載されない正当なレコードがある。そのため、
すべての protocol 設定に意味の薄い Application を強制することも適切でない。

## 決定

一つの Application は weblink なら protocol を持たず、federated / service なら作成時に確定した
一種類の protocol 設定を一つだけ持つ。後からの接続、解除、種別変更は提供しない。

OAuth2 client、SAML service provider、WS-Fed relying party は既存の protocol 固有 primary key を
維持し、nullable かつ unique な `application_id` を持つ。非 NULL の値は tenant と固定
protocol discriminator を含む複合外部キーで Application を参照する。これにより、同じ Application
への複数設定、異なる protocol table からの重複参照、tenant / 種別の不一致をデータベース制約で
拒否する。NULL は catalog 外設定を表す。

一括作成では protocol 設定をまず catalog 外 record として準備し、Application insert と
`application_id` relation の設定を一つの transaction で commit して catalog へ公開する。後半が
失敗しても意味のない Application は残らず、protocol record は正当な catalog 外設定として扱える。
Application 削除は関連 protocol 設定へ cascade する。Application に所有された protocol 設定を
低レベル管理 API から直接削除する操作は conflict として拒否する。

OAuth2 の物理テーブル名は汎用的な `clients` から `oauth2_clients` へ変更する。SAML と WS-Fed は
protocol 用語として一般的で既存 API / domain language と一致する
`saml_service_providers` / `wsfed_relying_parties` を維持する。protocol 中立に設計された
`provisioning_*` table も変更しない。

## 却下した代替案

- JSON binding 配列を維持する: 将来の柔軟性より、現在の参照整合性欠如と全件走査の負担が大きい。
- Application に nullable な protocol key 列を3本持たせる: catalog 外は表現できるが、型ごとの列と
  check constraint が上位 table に増え、新 protocol 追加時に Application schema が具体 key を知る。
- 汎用 `application_protocol_bindings` table を置く: RDB としては妥当だが、実在しない多対多と
  attach / detach lifecycle をモデル化し、今回の単一・不変という業務制約より自由度が高すぎる。
- protocol table の primary key を `application_id` に置き換える shared primary key:
  catalog 外設定を扱えず、既存の protocol identity と参照を全面的に変更する。
- すべての protocol 設定に Application を必須化する: DCR 等の catalog 外設定に意味の薄い
  Application record と lifecycle を強制する。
- 三つの table を `*_applications` に統一する: Application catalog と protocol 設定を同じ語で
  呼ぶため境界が曖昧になり、SAML / WS-Fed の標準的な domain language も失う。

## 影響

- 規範契約は `spec/contexts/application.yaml` の `models.ApplicationProtocolType`、
  `models.ApplicationProtocol`、`models.Application`、`interfaces.CreateAdminApplication` /
  `DeleteAdminApplication` を正とする。
- protocol 側の nullable relation と直接削除拒否は `spec/contexts/oauth2.yaml` の
  `models.OAuth2Client`、`spec/contexts/saml.yaml` の `models.SamlServiceProvider`、
  `spec/contexts/ws-federation.yaml` の `models.WsFedRelyingParty` と各 delete interface を正とする。
- binding 配列と attach / detach API は破壊的に削除し、管理 API は optional な単一 `protocol`
  projection を返す。
- 本機能は未リリースのため旧 Application データの backfill / dual-write は行わず、宣言的 schema
  と seed を最終形へ直接変更する。
- bounded context、module、global directory rule は変わらないため `ARCHITECTURE.md` の構造 map は
  変更しない。
