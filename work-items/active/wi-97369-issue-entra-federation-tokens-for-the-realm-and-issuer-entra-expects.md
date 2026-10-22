---
status: pending
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

## 対象範囲

- Entra の定型設定は、テナントに一つの RP（`wtrealm` と audience は `urn:federation:MicrosoftOnline`、返信先は `https://login.microsoftonline.com/login.srf`）を作り、その中にドメインと IssuerUri の組を持つ。構成の操作は組を追加または置き換える。
- Entra の RP へ発行するとき、利用者の UPN のドメインに一致する組の IssuerUri を発行者にする。一致する組がなければ発行を拒否する。
- 受動的な発行（REQ-WSFEDERATION-002）と能動的な発行（REQ-WSFEDERATION-004）の両方に同じ選び方を使う。
- REQ-WSFEDERATION-001 の構成の結果、PowerShell に渡す値、`ConfigureEntraFederation` の応答を改める。
- Entra の形の `wsignin1.0` と RST で、発行者、audience、NameID、UPN を Assertion から確かめる受け入れテストを足す。

## 対象外

- Hybrid Join の端末のクレーム（`accounttype`、`issuerid` など）。wi-98213 で扱う。
- 実テナントでの確認。wi-79 で扱う。

## 設計

発行者の選択は、定型設定の上の決定論的な計算にする。

```go
type EntraDomainFederation struct {
    Domain    string // 小文字にした検証済みドメイン
    IssuerURI string
}

// IssuerFor は、UPN のドメインに一致する組の IssuerUri を返す。一致しなければ ErrEntraDomainNotFederated。
func (p EntraFederationProfile) IssuerFor(upn string) (string, error)
```

ユースケースは、RP が Entra の定型設定を持つときだけ `IssuerFor` の結果で発行者を上書きし、それ以外の RP ではテナントの発行者のままにする。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| Entra の RP を一つにし、UPN のドメインで発行者を選ぶ | 採る | Entra が送る realm は一つだけなので、RP を分けても区別できない。AD FS の複数ドメインの構成と同じ形になる |
| ドメインごとに RP を分け、`wtrealm` で区別する（現在の形） | 採らない | Entra の要求はどのドメインでも同じ `wtrealm` で届く |
| 発行者をテナントの発行者に固定し、`issuerid` のクレームでドメインを示す | T001 で比べる | AD FS の複数ドメインの規則はこの形を使う。SAML 2.0 の文書は発行者の一致を求める。WS-Fed の Entra がどちらを照合するかを一次資料で確かめてから選ぶ |

## 計画

1. T001 で、WS-Fed の Entra が発行者と `issuerid` のどちらを照合するかと、利用者のトークンに要るクレームの集合（UPN、ImmutableID のクレーム `http://schemas.microsoft.com/LiveID/Federation/2008/05/ImmutableID` の要否）を、Microsoft の一次資料で確かめて決める。実装を始める前に決める。
2. 既存の Entra の RP（`wtrealm` が IssuerUri のもの）は、製品が未公開なので移行しない。

## タスク

- [ ] T001 [Plan] 発行者の照合とクレームの集合を一次資料で確かめ、設計の表の未決の行を決める。
- [ ] T002 [Spec] REQ-WSFEDERATION-001、002、004 と `ConfigureEntraFederation`、Entra の定型設定の設計（`relying-party.md`、`decisions.md`）を改める。
- [ ] T003 [Acceptance] Entra の形の `wsignin1.0` と RST が今は 400 になることを RED で確かめる。
- [ ] T004 [Domain] `IssuerFor` を単体テストで RED にしてから実装する。
- [ ] T005 [App] 受動的な発行と能動的な発行に発行者の選択を通し、受け入れテストを GREEN にする。
- [ ] T006 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`：WsFederation と、それを組み立てる HTTP の統合テスト。
- `mise run check`、`mise run check-spec-impact`：要件と TypeSpec の変更。
- ミューテーションテスト：`IssuerFor` と、発行者を上書きする分岐。

## リスク

- 発行者を誤ると、Entra は原因の分かりにくいエラーでサインインを拒否する。受け入れテストで Assertion の発行者と audience を直接確かめ、実テナントでの確認は wi-79 に残す。
