---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/directory-connector/features/runtime.md, requirement: REQ-DIRECTORYCONNECTOR-004 }
  - { path: docs/modules/sourcing/features/directory.md, requirement: REQ-SOURCING-016 }
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
---

# Directory Connector で取り込んだ AD の User に、Entra のフェデレーションが使う UPN と sourceAnchor を持たせる

## 動機

想定する構成では、Entra Connect が AD を Entra へ同期し、IdMagic は同じ AD を Directory Connector で取り込んで、Entra のドメインフェデレーションのサインインを担う。
この構成で、取り込んだ User へ Entra の定型設定のトークンを発行できない。

| 欠落 | 結果 |
| --- | --- |
| Connector は `objectGUID` を Entra Connect と同じ base64 にして外部の不変 ID として送るが、Sourcing はその値を対応（SourceCorrelation）にだけ保存し、User の属性に入れない | Entra の定型設定の sourceAnchor の属性が空になる。取り込んだ User は CSV のインポートでも変えられないので、値を入れる手段がない |
| AD の方言のデフォルトでは `preferred_username` が `sAMAccountName` であり、`@` を含まない | WsFederation は `preferred_username` を UPN として扱うので、発行者の IssuerUri を選べず、REQ-WSFEDERATION-002 の 500 で発行を拒否する |
| Entra の構成の検査は、UPN がフェデレーションするドメインに属する User だけを確かめる | `@` を含まない User は検査を素通りし、構成は成功してサインインの時点で初めて失敗する |

`ldap.attributes` で `preferred_username` を `userPrincipalName` に上書きすれば UPN だけは回避できるが、その案内はなく、sourceAnchor には回避の手段がない。

## 対象範囲

- Connector の AD の方言で、Entra の UPN にする値（`userPrincipalName`）と sourceAnchor にする値を User の項目に載せる。sourceAnchor のデフォルトは `objectGUID` の base64 とし、`ms-DS-ConsistencyGuid` も選べるようにする。
- Sourcing が、取り込んだ値を Entra の定型設定が参照できる User の属性に保存する。その属性は `unique` で `editable_by_user` でなく、取り込み元の管理下で変わる。
- Entra の構成の検査で、取り込み元の User のうちフェデレーションするドメインの UPN を持たない User を検出し、構成の結果に示す。
- Directory Connector と Entra の連携の設定を、運用者向けの文書に書く。

## 対象外

- 取り込んだ User のパスワードの確認。[[wi-11530-delegate-password-verification-to-active-directory-through-the-connector]] が扱う。
- Kerberos の principal と取り込んだ User の対応。[[wi-38381-map-kerberos-principals-to-ingested-directory-users]] が扱う。
- Entra Connect の同梱と、Entra 側の利用者の作成。
- 実テナントでの確認。[[wi-79-entra-real-tenant-end-to-end-verification]] が扱う。

## 設計

### 未解決の問い

着手時に決め、実装の前に利用者へ確認する。

- UPN の渡し方。
  - 案 A：AD の方言の `preferred_username` のデフォルトを `userPrincipalName` に変える。Entra と同じ名前でサインインする。既存の Connector の取り込みでは、利用者名が変わる。
  - 案 B：`preferred_username` は `sAMAccountName` のまま、UPN を別の属性に保存し、Entra の定型設定に UPN の属性を選ばせる（`DefaultEntraUPNAttribute` を設定にする）。
  - 製品は未公開なので、案 A でも移行は要らない。Entra と利用者名をそろえる点で案 A を推す。
- sourceAnchor を保存する属性の決め方。テナントが定義した属性を Connector の設定で指定するか、取り込み元が管理する組み込みの読み取り専用の属性にするか。
- 外部の不変 ID（対応のキー）と sourceAnchor を同じ値にするか。`ms-DS-ConsistencyGuid` を選ぶと、フォレスト間で移行した AD では `objectGUID` と食い違うので、対応のキーは `objectGUID` のまま分ける案を推す。
- 構成の検査で UPN を持たない User を見つけたとき、構成を拒否するか、結果に件数を示して通すか。フェデレーションしないドメインの User は正当に存在するので、取り込み元の User に限って示す案を推す。

## 計画

1. 未解決の問いを決め、DirectoryConnector、Sourcing、WsFederation の仕様を更新する。
2. Connector の写像、Sourcing の保存、WsFederation の構成の検査の順に実装する。
3. OpenLDAP では `objectGUID` を試せないので、AD の値の符号化は Connector の単体テストで、取り込みから発行までは Sourcing と WsFederation を組み合わせた受け入れテストで確かめる。

## タスク

- [ ] T001 [Plan] 未解決の問いを決める。
- [ ] T002 [Spec] REQ-DIRECTORYCONNECTOR-004、REQ-SOURCING-016、REQ-WSFEDERATION-001 を更新する。
- [ ] T003 [App] Connector の AD の方言で UPN と sourceAnchor を写す。
- [ ] T004 [App] Sourcing で sourceAnchor と UPN を User の属性に保存する。
- [ ] T005 [App] Entra の構成の検査で、UPN を持たない取り込み元の User を示す。
- [ ] T006 [Docs] Directory Connector と Entra の連携の設定を運用者向けの文書に書く。
- [ ] T007 [Verify] 取り込んだ User へ Entra の定型設定のトークンを発行し、ImmutableID と発行者を確かめる。

## 検証

- 受け入れテストで、Connector の項目として取り込んだ User に Entra の定型設定のパッシブサインインでトークンを発行し、ImmutableID が `objectGUID` の base64、発行者が UPN のドメインの IssuerUri であることを確かめる。
- Connector の単体テストで、`objectGUID` と `ms-DS-ConsistencyGuid` のバイト列を Entra Connect と同じ base64 にすることを確かめる。

## リスク

- sourceAnchor の値が Entra の `onPremisesImmutableId` と食い違うと、Entra はアカウントを重複させるか、サインインを無言で失敗させる。符号化は Entra Connect の値と突き合わせたテストで確かめる。
- 案 A を採ると、取り込み済みの User の利用者名が次の同期で変わる。未公開の間に決める。
