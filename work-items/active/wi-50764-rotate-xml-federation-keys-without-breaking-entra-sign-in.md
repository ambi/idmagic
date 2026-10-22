---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-23
priority: p2
depends_on: [wi-23994-return-entra-federation-settings-usable-with-microsoft-graph-powershell]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/signing-keys/features/separation.md, requirement: REQ-SIGNINGKEYS-006 }
---

# XML フェデレーションの鍵のローテーションで、Entra のサインインを止めない

## 動機

`XmlFederationSigning` の鍵のローテーションは、新しい鍵で直ちに署名を始め、古い鍵を 7 日の間メタデータに残す（REQ-SIGNINGKEYS-006）。
これは、メタデータを読み直す RP が古い鍵で署名された Assertion を検証し続けるための重なりである。
Entra は逆の向きの重なりを要する。

- Entra は、ドメインに設定した `signingCertificate` と `nextSigningCertificate` で Assertion を検証する（[internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation)）。
- Entra がフェデレーションのメタデータから新しい証明書を取り込むのは、現在の証明書の期限の 30 日前からである。
- idmagic の証明書の期限は 30 年（`keys_jose/certificate.go`）なので、自動の更新は実際には始まらない。

そのため、管理者が `XmlFederationSigning` の鍵をローテーションすると、その時点から Entra が知らない鍵で署名することになり、フェデレーションしたドメインの Microsoft 365 のサインインが止まる。
管理者が Entra の設定を更新するまで復旧しない。
管理画面とローテーションの API は、この影響を知らせない。
定期のバッチは `Signing` の鍵だけをローテーションするので、今は管理者の操作のときだけ起きる。

## 対象範囲

- 次の鍵を、署名に使う前にメタデータで公開する段階を設け、Entra の定型設定を持つテナントでは、その段階を経ずに署名の鍵を切り替えない。
- Entra の構成の結果で、次の鍵の証明書を `nextSigningCertificate` として渡せるようにする。
- REQ-SIGNINGKEYS-006 と鍵のライフサイクルの状態遷移を改める。

## 対象外

- SAML の SP ごとの鍵の扱いの見直し。SAML のプロファイルの鍵は、この段階を任意で使えるようにするだけにする。

## 設計

| 案 | 採否 | 理由 |
| --- | --- | --- |
| 次の鍵を先に公開し、一定期間の後か管理者の確認の後に署名の鍵へ切り替える | 候補 | Entra の `nextSigningCertificate` と、メタデータを読み直す RP の両方に合う。状態が一つ増える |
| 証明書の期限をローテーションの間隔に合わせ、Entra の自動の更新（期限の 30 日前からメタデータを読む）に任せる | 候補 | Entra の側の操作が要らない。Entra がどの URL のメタデータを読むかを確かめる必要がある |
| ローテーションの API で警告を出すだけにする | 採らない | 停止を防げない |

T001 で、Entra が自動の更新に使うメタデータの取得先と、`nextSigningCertificate` の運用の手順を一次資料で確かめてから選ぶ。

次の証明書を Entra に登録するため、先に wi-23994 で有効な証明書を含む Graph PowerShell の構成結果を整える。
事前公開の状態を追加するとき、この応答へ `NextSigningCertificate` を接続する。
Vault の提供元は再起動時に最新のバージョンを署名用に取り込むため、事前公開しただけの鍵を署名に使わない永続的な選択も、この設計で解決する。

## 計画

1. T001 で設計の候補を決める。
2. 状態遷移を変えるので、SigningKeyLifecycle の遷移の表とモデルテストを同時に改める。

## タスク

- [ ] T001 [Plan] Entra の証明書の更新の手順を確かめ、設計を決める。
- [ ] T002 [Spec] REQ-SIGNINGKEYS-006、状態遷移、Entra の定型設定の運用の記述を改める。
- [ ] T003 [Acceptance] Entra の定型設定を持つテナントで、ローテーションの直後の Assertion が、Entra に渡した証明書で検証できないことを RED で確かめる。
- [ ] T004 [App] 次の鍵の公開と切り替えを実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check`、`mise run check-spec-impact`。
- 状態遷移表のモデルテスト。

## リスク

- 状態を増やすと、JWT の鍵と XML の鍵で遷移が分かれる。用途ごとの違いを遷移の表に明記する。
