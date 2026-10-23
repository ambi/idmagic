---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: docs
affected_spec:
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-FED-REALM }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-FED-ISSUER }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-FED-NAMEID }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-MULTIDOMAIN-SUBDOMAIN }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-MULTIDOMAIN-ISSUER-URI }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-CLAIMS-USER }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-SOURCEANCHOR-VERBATIM }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-SOURCEANCHOR-AT-SIGN }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-GRAPH-PASSIVE-PROTOCOL }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-GRAPH-SIGNING-CERTIFICATE }
  - { path: docs/modules/ws-federation/standards.md, requirement: ENTRA-GRAPH-SIGN-OUT }
  - { path: docs/modules/ws-federation/standards.md, requirement: MSAL-WSTRUST-RST }
  - { path: docs/modules/ws-federation/standards.md, requirement: MSAL-WSTRUST-MEX }
  - { path: docs/modules/ws-federation/standards.md, requirement: WSFED-PASSIVE-SIGN-IN }
  - { path: docs/modules/ws-federation/standards.md, requirement: WSFED-SILENT-SIGN-IN }
  - { path: docs/modules/ws-federation/standards.md, requirement: WSTRUST13-ISSUE-BEARER }
  - { path: docs/modules/ws-federation/standards.md, requirement: WSTRUST13-WINDOWS-TRANSPORT }
  - { path: docs/modules/ws-federation/standards.md, requirement: WSS-USERNAME-TOKEN-PASSWORD }
  - { path: docs/modules/ws-federation/standards.md, requirement: WSADDRESSING-MESSAGE-ID-TO-ACTION }
primary_use_cases:
  - id: entra-interop-reference
    requirement: ENTRA-FED-REALM
    observable_result: Entra の realm の受動的な要求と RST に、Entra が受け入れる発行者、audience、ImmutableID、UPN を持つ Assertion が返る。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_federation_examples_test.go, name: TestEntraFederation_IssuesForTheRealmEntraSendsWithTheDomainIssuer, task: test-go-race }
    fault_model: 標準仕様の行が、実際の振る舞いを観測しないテストに結び付き、記録した要件と実装が食い違っても気付かない。
---

# Entra と Microsoft のクライアントとの相互運用の要件を、出典付きで WsFederation の標準仕様にまとめる

## 動機

WS-Federation、WS-Trust、Entra のドメインフェデレーション、Hybrid Join の作業で、相互運用の要件を毎回 Microsoft の文書や MSAL のソースから探し直した。
確かめた事実は作業項目の設計の節に散らばり、完了すると要約に縮められて読めなくなる。
一次資料で確かめていない思い込みのまま実装したことが、Entra とつながらない欠陥の原因になった（wi-97369、wi-16132、wi-14713）。

| 事実 | 一次資料 | 今の記録の場所 |
| --- | --- | --- |
| Entra は `wtrealm`、`AppliesTo`、audience に `urn:federation:MicrosoftOnline` を使い、発行者はドメインごとの IssuerUri と一致させる。サブドメインは親のドメインの IssuerUri を継ぐ。IssuerUri はドメインの間で重ならない | [SAML 2.0 の IdP の要件](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-fed-saml-idp)、[複数ドメインの構成](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-install-multiple-domains) | wi-97369 の設計 |
| 利用者のトークンのクレーム（UPN、ImmutableID、nameidentifier）と、端末のクレーム（`accounttype`、`onpremobjectguid`、`primarysid`、`issuerid`）、MFA のクレーム（`authnmethodsreferences` の `multipleauthn`） | [Entra Connect が設定する規則](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-azure-ad-trust)、[Hybrid Join の手動構成](https://learn.microsoft.com/en-us/entra/identity/devices/hybrid-join-manual)、[AD FS での MFA](https://learn.microsoft.com/en-us/entra/identity/authentication/howto-mfa-adfs) | wi-97369、wi-98213、wi-84807 の動機 |
| sourceAnchor は immutableId と同じで、大文字と小文字を区別し、文字列でない属性は base64 に、文字列の属性はそのまま使い、`@` を含めない | [Entra Connect の設計の概念](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-design-concepts) | wi-14713 の設計 |
| ドメインのフェデレーションの設定値（`PreferredAuthenticationProtocol`、`SigningCertificate`、`NextSigningCertificate`、`SignOutUri`、`FederatedIdpMfaBehavior`）と、証明書の自動の更新（期限の 30 日前からメタデータを読む） | [internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation) | wi-23994、wi-50764、wi-84807 の動機 |
| WS-Trust 1.3 の Issue の Action（`.../RST/Issue`）と RequestType（`.../Issue`） | [WS-Trust 1.3](http://docs.oasis-open.org/ws-sx/ws-trust/200512/ws-trust-1.3-os.html) 4.1 節 | 標準仕様の表（一部） |
| MSAL が送る RST の形と、MEX から方針、束縛、ポートの順にエンドポイントを選ぶ手順。`UsernameToken` を WS-SecurityPolicy 2005/07 の名前空間でだけ探す癖 | MSAL.NET の `WsTrust/MexDocument.cs`、`WsTrust/WsTrustEndpoint.cs`、`Internal/XmlNamespace.cs` | wi-16132 の設計 |
| Hybrid Join は WS-Trust 1.3 か 2005 の `windowstransport` を MEX で公開し、イントラネットにだけ公開する | [Hybrid Join の計画](https://learn.microsoft.com/en-us/entra/identity/devices/hybrid-join-plan)、[端末の登録の仕組み](https://learn.microsoft.com/en-us/entra/identity/devices/device-registration-how-it-works) | wi-98213 の動機 |

## 対象範囲

- 上の表の事実を、WsFederation の標準仕様の文書に、出典の URL と、対応の状態（`required`、`excluded` など）を付けた規範の行としてまとめる。
- 対応済みの行は、すでにある受け入れテストから `//spec:covers` で引く。
- 未実装の事実（MFA のクレーム、端末のクレーム、Hybrid Join の `windowstransport` の要件、証明書の自動の更新）は、今回は行にせず、担当する作業項目（wi-84807、wi-98213、wi-50764）が実装するときに行を足す。標準仕様の行はすべてテストから引く必要があり、今足すと未実装を固定するテストが要るからである。事実と出典は、その作業項目の動機にすでに書いてある。
- 既存の標準仕様の行の ID は、作業項目の `affected_spec` の形式（英大文字、数字、ハイフン）で引けないので、引ける形に改める。
- README の文書の索引から、まとめた節へ案内する。

## 対象外

- 規範の振る舞いそのものの変更。まとめるのは、今の実装と作業項目が拠っている事実である。
- 実テナントでの確認（wi-79）。
- Microsoft の文書の写し。出典の URL と、製品が拠る一文だけを書く。

## 設計

置き場所の候補を比べる。T001 で、文書の構成の検査（`check-spec` の文書の配置）と標準仕様の表の形式を確かめてから決める。

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| WsFederation の `standards.md` に、Entra のドメインフェデレーション、MSAL、Hybrid Join の節を足す | 外部の規範の一次情報の置き場所であり、規範の ID をテストと作業項目から引ける | Entra は標準ではなく製品なので、版の欄に文書の取得日を書くなど、表の形を合わせる必要がある |
| 研究ノート（`research` スキルの出力）として別の文書に置く | 調べた経緯と出典をそのまま残せる | 規範の文書でないので、テストと作業項目から引けず、現在の状態の一次情報が二つに分かれる |

### T001 の決定

| 問い | 決定 | 理由 |
| --- | --- | --- |
| 置き場所 | WsFederation の `standards.md` に、Microsoft の文書ごとの H2 の節を足す | 仕様の形式（[標準仕様](../../docs/formats/specification-format.md#4-標準仕様)）は、標準ごとに H2、出典の行、表を一つ置くと定める。出典の行の形は検査されないので、版の代わりに参照日を書ける |
| 行にする事実 | 実装済みで受け入れテストがある事実だけ | 形式はすべての ID をテストから引くことを求め、免除の一覧を置かない |
| ID の形 | 英大文字、数字、ハイフン。既存の行も改める | 作業項目の `affected_spec` は標準仕様の ID をこの形でだけ引ける。被覆の検査は宣言された ID をそのまま探すので、どちらの形でも引けるが、ファイルの中で形をそろえる |
| 検査ツールのテストの固定値（`WSFed-PassiveSignIn` など） | 変えない | ツール自身の入力の例であり、この文書の ID を参照していない |

事実は、この会話の前の作業項目（wi-97369、wi-16132、wi-14713、wi-23994）の着手時に、各出典を開いて確かめたものだけを書いた。

## 計画

1. T001 で置き場所と表の形を決め、既存の ID の改め方を決める。
2. 事実ごとに出典を開き直して確かめ、出典で確かめられなかった事実は書かない。
3. 対応済みの行を引く `//spec:covers` を既存のテストに足し、`check-spec` の被覆を通す。

## タスク

- [x] T001 [Plan] 置き場所、表の形、既存の ID の改め方を決めた。決定は設計の節に書いた。
- [x] T002 [Spec] 六つの節と 13 の行を足し、既存の 6 つの ID を改め、README の索引から案内した。行を足した直後の `check-spec` は、19 の ID のどれもテストが引かないことで失敗した。
- [x] T003 [Test] 既存の受け入れテスト 6 件に `//spec:covers` を足し、改めた ID を既存のテストと wi-98213 の記述で置き換えた。
- [x] T004 [Verify] 検証の節のとおり。

## 検証

- `mise run check-spec`（規範の参照と被覆）、`check-links`、`check-work-items`、`check-spec-impact`：成功。
- `mise run test-go-changed -- main`：1,191 件が成功、失敗とスキップは 0 件。変えたのはテストの注釈だけで、振る舞いは変えていない。
- `mise run check`：成功。

## リスク

- Microsoft の文書は予告なく改められる。各節に取得日を書き、実テナントの確認（wi-79）で食い違いを見つけたら改める。

## 要約

WsFederation の標準仕様に、Microsoft Entra の IdP の要件、複数のドメインのフェデレーション、Entra ID の信頼のクレームの規則、sourceAnchor、Microsoft Graph の internalDomainFederation、MSAL.NET の WS-Trust の六つの節と 13 の行を、出典と参照日を付けて足した。
どの行も、実装済みの振る舞いを観測する既存の受け入れテストから引く。
未実装の要件（MFA のクレーム、端末のクレーム、Hybrid Join、証明書の自動の更新）は、担当の作業項目が実装するときに行を足す。
既存の WS-Federation、WS-Trust、WS-Security、WS-Addressing の行の ID を、作業項目の `affected_spec` から引ける英大文字の形に改めた。
