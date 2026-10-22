---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-23
priority: p3
depends_on: [wi-97369-issue-entra-federation-tokens-for-the-realm-and-issuer-entra-expects, wi-16132-accept-the-ws-trust-issue-action-and-publish-a-mex-clients-can-read, wi-65-kerberos-spnego-inbound-silent-sso, wi-43000-sync-active-directory-computer-accounts-through-the-directory-connector]
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.WsTrustMetadataExchange }
---

# WS-Trust の windowstransport で、Hybrid Join の端末の登録に要るトークンを発行する

## 動機

フェデレーションしたドメインで Microsoft Entra hybrid join を使うには、STS が次を満たす必要がある（[Microsoft Entra hybrid join の手動構成](https://learn.microsoft.com/en-us/entra/identity/devices/hybrid-join-manual)、[端末の登録の仕組み](https://learn.microsoft.com/en-us/entra/identity/devices/device-registration-how-it-works)）。

| 要件 | 現在の状態 |
| --- | --- |
| Windows の端末が、統合 Windows 認証（コンピューターアカウントの Kerberos）で能動的な WS-Trust のエンドポイント（1.3 か 2005）を呼べる | `windowstransport` は実装しないと決めている（[判断](../../docs/modules/ws-federation/decisions.md#ws-trust-の能動的な対応を-usernamemixed-の-issue-だけに絞る)） |
| そのエンドポイントを MEX で公開する | MEX は usernamemixed だけを公開する |
| トークンに `accounttype`（`DJ`）、`onpremobjectguid`、`primarysid` を含め、複数の検証済みドメインでは `issuerid` も含める。利用者に ImmutableID のクレームを発行しているなら、コンピューターにも対応する ImmutableID を含める | コンピューターの主体とそのクレームがない |
| `windowstransport` はイントラネットにだけ公開し、外部へ公開しない | 公開の範囲を絞る設定がない |

Entra の定型設定は「Hybrid Join の端末の登録は提供しない」と案内している。
判断の再検討の条件（対象のクライアントが `windowstransport` を必須とするようになった場合）に当たるので、判断を改める。

## 対象範囲

- `/{realm}/trust/13/windowstransport` で、HTTP の `Negotiate` で認証した WS-Trust 1.3 の Issue を受ける。RST には UsernameToken を求めない。
- Kerberos のサービスチケットの検証は wi-65 の部品を使い、NTLM は受けない。
- チケットの主体がコンピューターアカウント（`<名前>$@<REALM>`）なら、wi-43000 のコンピューターアカウントに解決し、`accounttype=DJ`、`onpremobjectguid`、`primarysid`、`issuerid`（wi-97369 のドメインの IssuerUri）を持つ Assertion を Entra の RP へ発行する。無効か未知のコンピューターには発行しない。
- 主体が利用者なら、利用者のトークンに `accounttype=User` を足して、usernamemixed と同じクレームで発行する。
- MEX に、`http:NegotiateAuthentication` の方針を持つ `windowstransport` の束縛を足す。
- `windowstransport` を受ける発信元を、テナントが設定する社内の IP の範囲に限る。設定がなければエンドポイントを閉じる。
- `WSTrust13-WindowsTransport` を `required` に改め、能動的な STS の判断、Entra の定型設定の案内（`known_limitations`）、wi-65 と wi-79 の対象外の記述を改める。wi-79 の検証範囲を、Hybrid Join の登録の成功の確認に改める。

## 対象外

- WS-Trust 2005 の `windowstransport` と `certificatemixed`。Entra は 1.3 か 2005 のどちらかがあれば足りる。
- 端末の書き戻し（Entra Connect の device writeback）と、AD の Service Connection Point の設定。どちらも社内の AD 側の作業であり、運用の文書で案内する。
- Windows 7 と 8.1 の down-level の端末。

## 設計

能動的な STS の発行は、資格情報の種類ごとに主体を決め、あとは同じ発行の経路を通す。

```go
// ActivePrincipal は、能動的な STS が資格情報から決めた主体である。
type ActivePrincipal interface{ activePrincipal() }

type UserPrincipal struct{ User idmdomain.User }
type ComputerPrincipal struct {
    ObjectGUID []byte
    ObjectSID  string
    Domain     string
}

// BuildDeviceClaims は、Entra の定型設定とコンピューターの主体から端末のクレームを作る計算である。
func BuildDeviceClaims(profile EntraFederationProfile, computer ComputerPrincipal) ([]claimdomain.Claim, error)
```

| 論点 | 案 | 採否 | 理由 |
| --- | --- | --- | --- |
| 外部への公開の制限 | テナントの社内 IP の範囲に限り、未設定なら閉じる | 採る | Microsoft は外部へ公開しないよう求める。SaaS の idmagic には社内の専用の入口がないので、発信元で絞る |
| 外部への公開の制限 | 制限せず、Kerberos だけを受ける | 採らない | NTLM を受けなくても、外部からの総当たりの面が残る |
| コンピューターの値の出どころ | wi-43000 の同期した記録を引き、PAC の SID と照合する | 採る | `objectGUID` は PAC にない。同期した記録と PAC の SID が一致することを確かめて、取り違えを防ぐ |
| アプリケーションの関門 | コンピューターの主体には割り当てとサインインポリシーを適用せず、Entra の RP だけに発行する | T001 で決める | 割り当ては利用者を対象とする。コンピューターに適用する規則がないため、発行先を Entra の RP に限ることで代える案を検討する |

## 計画

1. T001 で、アプリケーションの関門の扱いと、コンピューターの ImmutableID の要否（wi-97369 の決定による）を決める。
2. Kerberos の検証、コンピューターの主体の解決、クレームの計算、エンドポイント、MEX の順に作る。
3. 単体と結合のテストは Kerberos の検証をポートの偽物で置き換え、実際の Kerberos の検証は wi-65 の結合テスト（KDC を持つテスト環境）に合わせる。端末からの登録の成功は wi-79 で確かめる。

## タスク

- [ ] T001 [Plan] 未決の論点を決め、判断（`decisions.md`）の改訂案を書く。
- [ ] T002 [Spec] 標準仕様の表、能動的な STS の要件、MEX、Entra の定型設定の案内を改める。
- [ ] T003 [Acceptance] コンピューターの Kerberos の主体による RST が、端末のクレームを持つ Assertion を返すことを RED で確かめる。社外の発信元、未知と無効のコンピューター、NTLM の拒否も含める。
- [ ] T004 [Domain] `BuildDeviceClaims` と発信元の範囲の判定を単体テストで RED にしてから実装する。
- [ ] T005 [App] エンドポイント、主体の解決、MEX の束縛を実装し、受け入れテストを GREEN にする。
- [ ] T006 [Docs] wi-65 と wi-79 の対象外と検証範囲、運用の文書（SCP、device writeback の案内）を改める。
- [ ] T007 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check`、`mise run check-spec-impact`。
- ミューテーションテスト：`BuildDeviceClaims`、発信元の範囲の判定、主体の種類による分岐。
- 実テナントでの登録の成功は wi-79 で確かめる。

## リスク

- コンピューターの主体の解決を誤ると、別の端末として Entra に登録される。PAC の SID と同期した記録の照合を受け入れテストで確かめる。
- 発信元の範囲の設定を誤ると、社外へ公開するか、社内の端末を拒否する。未設定を閉じる側に倒し、拒否を監査に残す。
