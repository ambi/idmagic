---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-23
priority: p3
depends_on: [wi-95-ldap-ad-user-federation]
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
---

# Directory Connector で Active Directory のコンピューターアカウントを取り込む

## 動機

Hybrid Join の端末の登録（wi-98213）では、端末を認証した STS が、コンピューターアカウントの `objectGUID`（`onpremobjectguid`）と `objectSid`（`primarysid`）をクレームとして発行する必要がある（[Microsoft Entra hybrid join の手動構成](https://learn.microsoft.com/en-us/entra/identity/devices/hybrid-join-manual)）。
Kerberos のチケットの PAC はコンピューターの SID とグループの SID を運ぶが、`objectGUID` は運ばない。
idmagic はコンピューターアカウントを一切持たず、wi-95 の Directory Connector もユーザーと最小限のグループだけを取り込む。

## 対象範囲

- Directory Connector が、AD のコンピューターアカウントの `objectGUID`、`objectSid`、`sAMAccountName`、`dNSHostName`、ドメイン、有効か無効かを取り込む。
- idmagic は、テナントのコンピューターアカウントを `objectSid` と、Kerberos の主体名（`<sAMAccountName>@<REALM>`）で引けるように保存する。
- AD で削除または無効化されたコンピューターアカウントを、次の同期で無効として扱う。
- 取り込みの要件を、Directory Connector の仕様に追加する。

## 対象外

- コンピューターアカウントの管理の画面と管理 API の書き込み。読み取りの API は、要るかを T001 で決める。
- Intune などの端末管理の情報。

## 設計

コンピューターアカウントの値の出どころを比べる。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| Directory Connector が同期する | 採る | wi-95 は、idmagic が AD の資格情報を持たず、Connector からの外向きの接続だけを使うと決めている。同じ経路で足りる |
| 発行のたびに idmagic が LDAP で引く | 採らない | idmagic が AD への到達経路と資格情報を持つことになり、wi-95 の信頼境界に反する |
| Kerberos の PAC だけから導く | 採らない | PAC は `objectGUID` を運ばない |

保存先のモジュールは、境界を選ぶ判断手順で決める。
候補は、User と同じく IdManagement の新しい Aggregate にする案と、Connector を持つモジュールに置く案である。
発行の側（WsFederation）は、所有者が公開する読み取りの操作だけを使う。

## 計画

1. T001 で保存先のモジュールを境界の判断手順で決め、記録をこの設計に書く。
2. wi-95 の Connector の同期の形（差分、削除の検出）をそのまま使い、コンピューターアカウントの種類を足す。
3. `affected_spec` は、取り込んだ値を使う能動的な STS の要件を指す。T002 で Directory Connector の要件を追加したら、その要件に差し替える。

## タスク

- [ ] T001 [Plan] 保存先のモジュールと読み取りの API の要否を決め、境界の判断の記録を書く。
- [ ] T002 [Spec] Directory Connector の仕様に、コンピューターアカウントの取り込みの要件を追加する。
- [ ] T003 [Acceptance] Connector の同期で、コンピューターアカウントが `objectSid` と主体名で引けることを RED で確かめる。
- [ ] T004 [App] 保存、同期、無効化を実装し、メモリと PostgreSQL の共有の契約テストを通す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check-boundaries`、`mise run check`。

## リスク

- コンピューターアカウントの数はユーザーより多いことがある。同期の量と索引を T001 で見積もる。
