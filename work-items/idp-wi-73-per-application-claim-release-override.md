---
id: idp-wi-73-per-application-claim-release-override
title: "アプリケーション単位で claim release を上書きする"
created_at: 2026-06-27
authors: ["tn"]
status: pending
risk: medium
---

# Motivation
Okta も Entra ID も、アプリケーションごとに「そのアプリに渡す属性」を個別設定できる。
同じユーザーでも、ある SP には employeeNumber と部署だけ、別の RP にはメールと表示名だけ、
というように出力する claim をアプリ単位で絞る/写像する。最小権限と属性最小化の観点で重要で、
プロトコル (OIDC claim / SAML AttributeStatement / WS-Fed claim URI) をまたいで効く。

idmagic は ADR-059 / [[wi-63-federation-metadata-and-claims-mapping]] で宣言的・fail-closed な
ClaimMappingPolicy を持ち、ADR-064 で `ClaimMapping` を protocol-neutral capability として
分離した。ただし現状の release policy はテナント/プロトコル既定の粒度で、アプリ単位の上書きが
ない。本 WI は [[wi-69-application-catalog-aggregate-and-assignment]] の Application に
claim release の上書きをぶら下げ、ClaimMapping が解決時にアプリ別ルールを適用できるようにする。

# Scope
- **decision**:
  - ADR-059 への追補 (または軽量 ADR-069): claim release 解決の優先順位を確定する。 テナント既定 → Application 上書き → (将来 assignment 単位) の合成順序、未定義属性の fail-closed 維持、上書きが各 protocol の wire projection (OIDC/SAML/WS-Fed) に どう反映されるかを決める。所有は ClaimMapping、アプリ紐付けは Application。
- **scl**:
  - ClaimMapping に ApplicationClaimMappingPolicy (Application を参照する ClaimMappingPolicy の 上書きセット) を追加し、IssuedClaim 解決時に合成する。
  - interface: 管理者の Application 配下 claim release CRUD。解決自体は既存の claim 発行経路 (UserInfo / ID Token / SAML AttributeStatement / WS-Fed) に組み込む。
  - アプリ別上書きは属性 claim だけでなく subject 識別子も対象にする。ClaimMapping の NameIdConfiguration をアプリ単位で写像し、OIDC `sub` / SAML NameID / WS-Fed の 主体識別子を Application ごとに決められるようにする (下流 SP がメール以外でアカウントを 突き合わせるケース)。per-user の literal override は SWA / password-vault 導入時の将来 WI。
  - event: ApplicationClaimMappingUpdated。
  - invariant: ClaimMappingFailClosed の維持 (上書きでも未許可属性は出さない)、 ClaimMappingResolvedPerApplication (アプリ別ルールが全 protocol projection に効く)。
  - permission: 既存 AdminApplicationsManage を再利用、または AdminApplicationClaimsManage を追加。
- **go**:
  - per-application claim release policy の persistence (application_claim_release テーブル、tenant scope)。
  - claim 解決器を「テナント既定 + アプリ上書き」の合成にし、OAuth2 / WsFederation / 将来 Saml の発行経路から共通に呼ぶ。
- **http**:
  - /admin/applications/{id}/claim-release の取得/更新。
- **ui**:
  - admin: Application 詳細に claim release エディタ (出力属性の選択・写像・既定からの差分表示)。

# Out of Scope
- 属性変換式言語 (任意の式評価) の導入。初期は選択 + 単純写像 + 既定差分のみ。
- assignment (user/group) 単位のさらに細かい上書き。アプリ単位を先に入れ、後続で検討。
- Application 本体・割当 ([[wi-69-application-catalog-aggregate-and-assignment]])。

# Verification
- `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `bun run yaml-check:work-items` (in: tools)
- `bun run yaml-check:scl` (in: tools)
- 手動: 同一ユーザーで 2 つのアプリに別々の claim release 上書きを設定し、それぞれの ID Token / アサーションに出力属性が想定どおり差分で現れることを確認する。未許可属性が 上書きでも漏れないことを確認する。

# Risk Notes
claim 解決の合成順序を誤ると属性過剰開示や fail-closed の破れにつながる。ADR-059 の
宣言的・fail-closed 原則を維持したまま「テナント既定 + アプリ上書き」を合成し、解決器は
全 protocol 発行経路で共通の 1 経路に保つ。上書きは出力を絞る方向を既定とし、新規属性の
追加は明示許可済みソースに限る。
