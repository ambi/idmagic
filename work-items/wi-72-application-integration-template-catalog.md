---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-27
---

# アプリ統合テンプレートカタログ (gallery) からアプリケーションを作成する

## Motivation
Okta Integration Network も Entra ID のアプリギャラリーも、よく使われる SaaS の
「事前定義テンプレート」を多数持つ。管理者はゼロからプロトコル設定を組むのではなく、
テンプレートを選ぶだけで、推奨 protocol binding 種別、既定の claim mapping、アイコン、
必須メタデータ (ACS URL の入力欄、entityID の形など) が用意された状態でアプリを作成できる。
これが SSO 導入の摩擦を大きく下げている。

[[wi-69-application-catalog-aggregate-and-assignment]] で Application を手で作れるようになるが、
毎回 binding と claim を一から設定するのは煩雑で誤りやすい。本 WI は Application に
テンプレートカタログを追加し、テンプレートから Application を instantiate できるようにする。
デモ IdP として「既知アプリを数クリックで接続する」現代 IdP の体験を示す。

## Scope
- **decision**:
  - 新規 ADR-068: テンプレートの所有と供給形態を確定する。テンプレートは Application が所有する read-only カタログ (リポジトリ同梱の宣言データ) とし、 テナント横断の共有定義と、テナント独自テンプレート登録を許すかを決める。テンプレートが 規定する内容 (推奨 binding type、既定 claim release、icon、必須入力フィールドの schema、 ベンダー metadata) と、instantiate 時に Application へコピーする範囲を決める。
- **scl**:
  - Application に ApplicationTemplate / TemplateProtocolDefault / TemplateClaimDefault / TemplateInputField を追加する。
  - interface: ListApplicationTemplates / GetApplicationTemplate / CreateApplicationFromTemplate (テンプレ + 入力値から Application + binding を生成)。
  - event: ApplicationCreatedFromTemplate。
  - permission: 既存 AdminApplicationsManage を再利用 (新規作成の一種)。
- **go**:
  - 同梱テンプレートデータのロードと検証 (tenant 非依存の read-only セット)。
  - instantiate ロジック: テンプレートの既定値 + 管理者入力を検証し、Application と protocol binding と既定 claim release を生成する (wi-69 の aggregate を再利用)。
- **http**:
  - /admin/application-templates の一覧/詳細と、テンプレートからの作成エンドポイント。
- **ui**:
  - admin: テンプレートギャラリー (検索・アイコン表示) と、選択 → 入力 → 作成のウィザード。
- **documentation**:
  - README にテンプレート定義の追加方法と同梱例を書く。

## Out of Scope
- 外部レジストリ (OIN / Entra gallery) からのオンライン取り込み。初期は同梱テンプレートのみ。
- テンプレートごとの SCIM provisioning 既定値 ([[wi-31-scim2-provisioning]] / [[wi-45-outbound-scim-provisioning]] 実装後に拡張)。
- SAML SP テンプレート本体 ([[wi-29-saml2-idp]] 実装後に binding 種別を追加)。
- Application 本体・割当 ([[wi-69-application-catalog-aggregate-and-assignment]])。

## Plan
- [[ADR-064-protocol-contexts-and-application-catalog]] と [[ADR-066-application-as-single-editor-surface]] に従い、template は Application Catalog の seed capability とし、OAuth2/SAML/WS-Fed の別々の作成 UI を復活させない。
- catalog entry は stable key/version、表示 metadata、application kind、protocol binding blueprint、入力 field schema/default/validation、documentation URL を持つ。secret、tenant ID、実 client/RP ID は含めない。
- catalog 自体は versioned static artifact から開始し、tenant override/marketplace は scope 外にする。template 更新は既に作成済み Application を自動変更せず、作成時の template key/version を provenance として記録する。
- instantiate は field validation→Application create→protocol binding provisioning を既存 application provisioning use case の transaction/compensation semantics で実行し、OIDC secret は既存契約どおり一度だけ返す。
- UI は browse/search/category/details/configure/review の wizard とし、protocol advanced editor は作成後の Application editor に委ねる。

## Tasks
- [ ] T001 [SCL] ApplicationTemplate/FieldSchema/BindingBlueprint、List/Get/Instantiate interfaces、provenance/constraints/contracts/scenarios を追加して再生成する。
- [ ] T002 [Catalog] versioned static catalog format、schema validator、duplicate key/version check と初期 OIDC/SAML/WS-Fed/weblink/service entries を追加する。
- [ ] T003 [Usecase] template input を既存 CreateApplication/ProvisionBinding command に変換し、失敗時 compensation と一回 secret response を実装する。
- [ ] T004 [HTTP] list/search/get/instantiate endpoint、category/filter、template version conflict error を追加する。
- [ ] T005 [UI] gallery/detail/configure/review wizard と作成後 Application editor への遷移、一回 secret 表示を実装する。
- [ ] T006 [Verify] 全 template の schema fixture、version provenance、invalid field、protocol provisioning rollback、tenant permission と snapshot/accessibility を検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- `just yaml-check-work-items`
- `just yaml-check-scl`
- 手動: テンプレートを選び必須入力を埋めて Application を作成し、生成された OIDC binding と 既定 claim release が正しく、そのまま SSO できることを確認する。

## Risk Notes
wire behavior は変えず、既存 Application aggregate の上に作成体験を足す機能。主なリスクは
テンプレートの既定値が誤った binding / claim を生むことなので、instantiate は手動作成と
同じ検証経路を必ず通す。テンプレートデータは read-only に保ち、テナント書き込みと分離する。
