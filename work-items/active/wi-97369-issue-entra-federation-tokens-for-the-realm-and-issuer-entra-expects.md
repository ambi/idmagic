---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: docs/modules/ws-federation/features/passive-sign-in.md, requirement: REQ-WSFEDERATION-002 }
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.EntraFederationProfile }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.EntraDomainFederation }
primary_use_cases:
  - id: entra-passive-sign-in
    requirement: REQ-WSFEDERATION-002
    observable_result: Entra が送る wtrealm の受動的な要求に、利用者のドメインの IssuerUri を発行者とし、Entra を audience とし、ImmutableID と UPN を持つ署名した Assertion が login.srf へ返る。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_federation_examples_test.go, name: TestEntraFederation_IssuesForTheRealmEntraSendsWithTheDomainIssuer, task: test-go-race }
    fault_model: Entra の要求が登録済みの RP に解決できないか、発行者がドメインの IssuerUri でないか、ImmutableID の公開の検査で発行が拒否される。
  - id: entra-active-sign-in
    requirement: REQ-WSFEDERATION-004
    observable_result: Entra の AppliesTo の RST に、利用者のドメインの IssuerUri を発行者とする同じ形の Assertion を持つ RSTR が返る。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_federation_examples_test.go, name: TestEntraFederation_IssuesForTheRealmEntraSendsWithTheDomainIssuer, task: test-go-race }
    fault_model: 能動的な発行だけが発行者の選択を通らず、テナントの発行者で発行する。
---

# Entra のドメインフェデレーションに、Entra が送る realm で応じ、ドメインの IssuerUri を発行者にする

## 動機

Entra のドメインフェデレーションの定型設定（`EntraFederationProfile`）は、実際の Entra とつながらない形になっている。

| 項目 | Entra が求めるもの | 現在の実装 |
| --- | --- | --- |
| 受動的な要求の `wtrealm`、能動的な要求の `AppliesTo` | `urn:federation:MicrosoftOnline`（Entra 自身の識別子） | RP の `wtrealm` を IssuerUri（省略時は `urn:idmagic:entra:<ドメイン>`）にする。Entra の要求は登録済みの RP に解決できず、400 で拒否される |
| Assertion の audience | `urn:federation:MicrosoftOnline` | IssuerUri |
| Assertion の発行者 | そのドメインに設定した IssuerUri。複数のドメインがあれば、ドメインごとの値 | テナントの発行者（`support.RequestIssuer`）。PowerShell の案内が Entra に設定させる IssuerUri と一致しない |
| 複数の検証済みドメイン | 一つの RP（Entra）に対し、利用者の UPN のドメインで発行者を選ぶ | ドメインごとに `wtrealm` の異なる RP を作る。Entra はどのドメインでも同じ `wtrealm` を送るので、区別できない |

一次資料：Microsoft Entra の [SAML 2.0 の IdP の要件](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-fed-saml-idp)は、Audience を `urn:federation:MicrosoftOnline`、Issuer を「ドメインごとに設定した URI と一致する値」と定める。
既存のテストは `urn:idmagic:entra:contoso.com` を `wtrealm` とする RP の保存だけを確かめており、Entra の形の要求で発行まで通す経路を観測していない。

着手時の受け入れテストで、さらに二つの欠陥を確かめた。

- 現在の `wtrealm`（IssuerUri）で要求しても、発行は 500（`claim issuance failed`）になる。NameID の元にする合成の属性 `entra_immutable_id` がテナントの属性の定義にないので、クレームの公開の検査が「公開できない属性」として拒否する。Entra の定型設定は、どの経路でもトークンを発行できなかった。
- 別のドメインの構成に、既存のドメインと同じ IssuerUri を指定すると、拒否せずに既存のドメインの RP を置き換える。Entra は同じ IssuerUri を二つのドメインに設定できない。

## 対象範囲

- Entra の定型設定は、テナントに一つの RP（`wtrealm` と audience は `urn:federation:MicrosoftOnline`、返信先は `https://login.microsoftonline.com/login.srf`）を作り、その中にドメインと IssuerUri の組を持つ。構成の操作は組を追加または置き換える。
- Entra の RP へ発行するとき、利用者の UPN のドメインに一致する組の IssuerUri を発行者にする。サブドメインの UPN は親のドメインの組に一致させる。一致する組がなければ発行を拒否する。
- 受動的な発行（REQ-WSFEDERATION-002）と能動的な発行（REQ-WSFEDERATION-004）の両方に同じ選び方を使う。
- ImmutableID を、NameID と `http://schemas.xmlsoap.org/claims/nameidentifier` に加え、`http://schemas.microsoft.com/LiveID/Federation/2008/05/ImmutableID` のクレームでも発行する。
- 合成の属性 `entra_immutable_id` に、sourceAnchor の属性と同じ公開の可否を与える。
- 別のドメインと同じ IssuerUri の構成と、構成済みのドメインがあるのに sourceAnchor の属性を変える構成を拒否する。
- REQ-WSFEDERATION-001 の構成の結果、PowerShell に渡す値、`ConfigureEntraFederation` の応答、管理画面を改める。

## 対象外

- Hybrid Join の端末のクレーム（`accounttype`、`issuerid` など）。wi-98213 で扱う。
- 実テナントでの確認。wi-79 で扱う。
- ドメインを一つずつ外す操作。Entra の RP の削除は、すべてのドメインを外す。
- PowerShell の項目の名前と証明書の値。wi-23994 で扱う。
- sourceAnchor の構成の時点の検査の範囲と、構成の後の一意性。wi-14713 で扱う。

## 設計

### T001 の決定

| 問い | 決定 | 根拠 |
| --- | --- | --- |
| Entra がドメインを見分けるもの | Assertion の発行者を、ドメインの IssuerUri にする。`issuerid` のクレームは発行しない | [複数ドメインの構成](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-install-multiple-domains)は、AD FS が `issuerid` の規則で「トークンの発行者」をドメインの IssuerUri にし、一致しなければ認証が失敗すると述べる。SAML 2.0 の IdP の要件も、発行者のドメインごとの一致を求める |
| サブドメイン | UPN のドメインに、構成したドメインのうち最も長く一致する親のドメインの組を使う | 同じ文書は、サブドメインが親のドメインの設定を継ぐので、IssuerUri も親と一致させる必要があると述べる |
| 利用者のトークンのクレーム | UPN、ImmutableID のクレーム、永続的な NameID | [Entra Connect が設定する規則](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-azure-ad-trust)は、UPN、ImmutableID、nameidentifier を発行する。`accounttype` と `issuerid` は端末の登録と複数ドメインの AD FS の仕組みであり、端末は wi-98213 で扱う |
| 同じ IssuerUri の二つのドメイン | 400 の `invalid_request` で拒否する | 同じ文書は、Entra が同じ IssuerUri を二つのドメインに許さないと述べる |
| sourceAnchor の属性 | テナントの Entra の RP に一つ。構成済みのドメインがある間に別の属性を指定したら拒否する | 属性を変えると、構成済みのドメインの利用者の ImmutableID が変わり、Entra の利用者と結び付かなくなる |

### 型と操作

```go
type EntraDomainFederation struct {
    Domain    string `json:"domain"` // 小文字にした検証済みドメイン
    IssuerURI string `json:"issuer_uri"`
}

type EntraFederationProfile struct {
    Domains               []EntraDomainFederation
    SourceAnchorAttribute string
    ImmutableIDAttribute  string
    PassiveLogOnURI, ActiveLogOnURI, MetadataExchangeURI string
}

// IssuerFor は、UPN のドメインに最も長く一致する組の IssuerUri を返す。一致しなければ ErrEntraDomainNotFederated。
func (p EntraFederationProfile) IssuerFor(upn string) (string, error)

// WithDomain は、組を追加または置き換えた定型設定を返す。別のドメインと同じ IssuerUri なら ErrEntraIssuerInUse。
func (p EntraFederationProfile) WithDomain(domain EntraDomainFederation) (EntraFederationProfile, bool, error)

// EntraAttributeDefs は、sourceAnchor の属性が公開できるときだけ、合成の ImmutableID の属性を公開できる定義に加える。
func EntraAttributeDefs(defs []userdomain.UserAttributeDef, profile *EntraFederationProfile) []userdomain.UserAttributeDef
```

受動的な発行と能動的な発行のユースケースは、RP が Entra の定型設定を持つときだけ `IssuerFor` の結果を発行の結果に載せる。
アダプターは、結果の発行者が空でなければその値を、空ならテナントの発行者を Assertion に使う。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| Entra の RP を一つにし、UPN のドメインで発行者を選ぶ | 採る | Entra が送る realm は一つだけなので、RP を分けても区別できない。AD FS の複数ドメインの構成と同じ形になる |
| ドメインごとに RP を分け、`wtrealm` で区別する（現在の形） | 採らない | Entra の要求はどのドメインでも同じ `wtrealm` で届く |
| 発行者をテナントの発行者に固定し、`issuerid` のクレームでドメインを示す | 採らない | Microsoft の文書は、トークンの発行者がドメインの IssuerUri と一致することを求めている |
| 合成の属性を公開の検査から外す | 採らない | sourceAnchor が非公開の属性でも、その値が ImmutableID として漏れる |

## 計画

1. 既存の Entra の RP（`wtrealm` が IssuerUri のもの）は、製品が未公開なので移行しない。
2. 仕様、受け入れテストの RED、ドメインの計算、ユースケースとアダプター、管理画面の順に進める。

## タスク

- [x] T001 [Plan] 発行者の照合とクレームの集合を一次資料で確かめ、設計の表の未決の行を決めた。
- [x] T002 [Spec] REQ-WSFEDERATION-001、002、004、`ConfigureEntraFederation`、`EntraFederationProfile`、`EntraDomainFederation`、Entra の定型設定の設計（`relying-party.md`、`decisions.md`）を改めた。`check-spec` と `check-published-api-compat` は成功した。
- [x] T003 [Acceptance] `TestEntraFederation_*`（`entra_federation_examples_test.go`）を足し、RED を確かめた。Entra の `wtrealm` と `AppliesTo` は 400 `unknown relying party`、同じ IssuerUri の別のドメインの構成は 200 で既存の RP を置き換えた。現在の `wtrealm` でも 500 `claim issuance failed` になることを一時的なテストで確かめた。
- [x] T004 [Domain] `IssuerFor`、`WithDomain`、`EntraAttributeDefs` と ImmutableID のクレームの規則を実装した。単体テストは実装と同じ手順で書き、型がないことによるコンパイルの失敗を RED の代わりにした。
- [x] T005 [App] パッシブサインインと WS-Trust に共通する `issueForRelyingParty` で、ImmutableID、発行者、公開の定義を決め、受け入れテストを GREEN にした。構成の操作は `WithDomain` で組を足し、同じ IssuerUri と sourceAnchor の属性の変更を 400 で拒否する。
- [x] T006 [UI] 管理画面の一覧に、一つの Entra の RP のドメインと IssuerUri を並べた。`AdminEntraFederationPage.test.tsx` で二つのドメインの表示を確かめた。
- [x] T007 [Verify] 変更を検証した。結果は検証の節に書いた。

## 検証

- `mise run test-go-changed -- main`：851 件が成功、失敗とスキップは 0 件。WsFederation の PostgreSQL の契約テスト（JSONB に保存する定型設定の形が変わる）を含む。
- `mise run typecheck-ui`、`lint-ui`、`format-check-ui`、`test-ui-unit-file`（`AdminEntraFederationPage.test.tsx` の 4 件）：成功。Entra の画面を通る e2e テストはないので、e2e は選ばない。
- `mise run check`、`lint-go`、`check-spec-impact`、`check-boundaries`、`check-operation-references`：成功。
- ミューテーションテスト（変更行、`main` と比較）：`wsfederation/domain` は 14 件のうち 11 件を検出した。sourceAnchor の属性の既定値への置き換えが観測されていなかったので、指定した属性で判定するテストを足した。残る 3 件は、ローカル部が空の UPN、同じ長さの二つの構成済みドメイン（ドメインは重複しない）、スライスの初期容量で、観測する意味がない。`handlers_http` は 100%。
- 手動の故障注入：`usecases` は自身のテストを持たないので、受動的な発行と能動的な発行の発行者を落とす、合成の属性の公開の定義を外す、フェデレーションしていないドメインの拒否を外す、の四つを注入し、`TestEntraFederation_*` がすべて検出した。

## リスク

- 発行者を誤ると、Entra は原因の分かりにくいエラーでサインインを拒否する。受け入れテストで Assertion の発行者と audience を直接確かめ、実テナントでの確認は wi-79 に残す。

## 要約

REQ-WSFEDERATION-001、002、004 と、`ConfigureEntraFederation`、`EntraFederationProfile` を改め、`EntraDomainFederation` を足した。
Entra の定型設定は、テナントに一つの `urn:federation:MicrosoftOnline` の RP にドメインと IssuerUri の組を持ち、利用者の UPN のドメインに最も長く一致する組の IssuerUri を発行者にする。
受動的な発行と能動的な発行は同じ規則を使い、どのドメインにも属さない利用者には 500 で発行を拒否する。
ImmutableID は ImmutableID のクレームでも発行し、合成の属性は sourceAnchor の属性を公開できるときだけ公開できる。
着手時に、現在の定型設定が公開の下限でどの経路でも発行できなかったことと、同じ IssuerUri の別のドメインの構成が既存の RP を置き換えることを確かめ、両方を直した。
`issuerid` のクレームは発行しない。Microsoft の文書はトークンの発行者がドメインの IssuerUri と一致することを求めるからである。
ドメインを一つずつ外す操作は対象外とし、実テナントでの確認は wi-79 に残る。
