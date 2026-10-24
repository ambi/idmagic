---
status: pending
authors: [tn]
risk: critical
created_at: 2026-10-24
priority: p3
depends_on: [wi-11530-delegate-password-verification-to-active-directory-through-the-connector]
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007 }
---

# AD のパスワードを IdMagic でも確かめられるようにするかを決め、採るなら委譲の成功時に取り込む

## 動機

[[wi-11530-delegate-password-verification-to-active-directory-through-the-connector]] の認証の委譲では、AD から取り込んだ User のパスワードを、サインインのたびに Connector が AD で確かめる。
Connector か AD に届かない間、その User はパスワードでサインインできない。
AD のパスワードを IdMagic でも確かめられれば、この間もサインインを続けられる。

ただし、費用対効果は低いと見込む。
委譲を正とし、Connector の冗長化（[[wi-86455-run-several-directory-connector-agents-for-one-identity-source]]）で停止の時間を短くすれば足りる組織が多いからである。
また、どの方式も IdMagic にパスワードを確かめるための値を持たせ、AD とのずれと漏えいの面を増やす。
したがって、**実装しないことを最も有力な結論**として検討し、実装しない場合はその判断を要約に書いて中止する。

## 対象範囲

- 実装するかどうかを、需要、委譲と冗長化で足りるか、増えるリスクから決める。
- 実装する場合は、第一の案として、委譲したサインインを AD が成功と答えたときに、IdMagic のパスワードのハッシュとして保存する方式を設計する。
- 実装する場合は、IdMagic に保存した値を、委譲が使えないときの代わりの確認にだけ使う条件と、AD での変更や無効化とのずれを閉じる規則を決める。

## 対象外

- AD の複製のプロトコル（MS-DRSR）で NT ハッシュを読む方式（Entra の Password Hash Sync と同じ方式）。サービスアカウントに Replicating Directory Changes All の権限が要り、これは DCSync 攻撃と同じ権限なので採らない。
- Entra ID のパスワードの取り込み。Entra ID はパスワードもハッシュも API で出さないので、できない。Entra の User は上流の IdP との連携でサインインさせる。

## 設計

### 比べる案

| 案 | 仕組み | 長所 | 短所 | 位置付け |
| --- | --- | --- | --- | --- |
| 0. 実装しない | 委譲を正とし、停止は冗長化で短くする | 新しい秘密を持たず、AD とのずれも生まれない | Connector と AD に届かない間は、委譲した User がサインインできない | 最も有力 |
| 1. 委譲の成功時に取り込む | AD が正しいと答えたパスワードを、IdMagic のパスワードのハッシュとして保存する | AD の権限を増やさず、Connector の変更も要らない | 一度もサインインしていない User には効かない。AD で変えた後も、古いパスワードが IdMagic で通る期間ができる | 実装するなら第一の案 |
| 2. DC へのパスワードフィルター | 全 DC にフィルターの DLL を入れ、変更の時点で平文を捕まえて送る（Google の GCDS 用の Password Sync、MIM の PCNS と同じ方式） | AD で変えたパスワードがすぐ届き、ずれが小さい | 全 DC へのネイティブの DLL の導入、署名、更新の負担が大きい。導入の前からあるパスワードは届かない。DC の上で平文を扱う | 案 1 のずれが許されない需要が出たときの候補として残す |

### 未解決の問い

- 案 1 を採る場合、IdMagic に保存した値を使うのは委譲が使えないときだけか、常に先に使うか。常に先に使うと、AD で変えた古いパスワードが通り続ける。
- AD 側の無効化、ロック、パスワードの変更を、Directory Connector の取り込み（`userAccountControl`、`pwdLastSet`）から知り、保存した値を捨てる規則。

## 計画

1. 案 0 を結論の候補として、需要とリスクを確かめ、利用者と決める。
2. 案 0 なら、判断を要約に書いて中止する。
3. 案 1 か案 2 なら、要件の差分と脅威モデルを設計に書いてから実装する。

## タスク

- [ ] T001 [Plan] 実装するかどうかと、採る案を決める。
- [ ] T002 [Spec] 採る場合は Authentication と Sourcing の要件を書く。
- [ ] T003 [App] 採る場合は実装する。
- [ ] T004 [Verify] 採る場合は変更を検証する。

## 検証

- 採る場合は、委譲の成功時の保存、Connector の停止中の代わりの確認、AD でのパスワードの変更の後に古いパスワードが通らなくなる時点を E2E で確かめる。

## リスク

- IdMagic に保存した値が漏れると、AD のパスワードへの総当たりの材料になる。保存のハッシュの方式は、IdMagic の既存のパスワードと同じ強さにする。
- AD とのずれの間、AD で無効にしたつもりのパスワードが IdMagic で通る。
