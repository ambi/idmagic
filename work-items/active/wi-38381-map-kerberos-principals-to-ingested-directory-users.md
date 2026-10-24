---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/negotiate.md, requirement: REQ-AUTHENTICATION-042 }
  - { path: docs/modules/sourcing/features/directory.md, requirement: REQ-SOURCING-016 }
---

# Directory Connector で取り込んだ AD の User を、Kerberos の principal に対応付ける

## 動機

Kerberos による無音のサインイン（REQ-AUTHENTICATION-042）は、運用者が設定ファイルに書いた principal の名前から User ID への対応だけを使う。
Directory Connector で AD から取り込んだ User は、この対応と連動しない。
ドメインに参加した PC から Microsoft 365 などへ無音でサインインさせるには、運用者が利用者の人数分の対応を手で書き、入退社のたびに直す必要がある。

## 対象範囲

- Connector の AD の方言で、Kerberos の principal の名前にする値（`sAMAccountName`）を User の項目に載せる。
- Connector の設定またはテナントの設定で、Connector と Kerberos の realm を結び付ける。
- Authentication が、検証した principal を、その realm の Connector から取り込んだ有効な User へ対応付ける。設定ファイルの明示的な対応は残し、先に評価する。
- 取り込みで無効にした User と削除を予約した User は、対応付けても拒否する。

## 対象外

- 取り込んだ User のパスワードの確認（[[wi-11530-delegate-password-verification-to-active-directory-through-the-connector]]）。
- 複数のフォレストと信頼関係のある realm。
- Entra の UPN と sourceAnchor（[[wi-69652-pass-upn-and-source-anchor-of-ad-users-to-entra-federation]]）。

## 設計

### 未解決の問い

- 対応の索引を Sourcing の対応（SourceCorrelation）に持たせるか、User の属性にするか。Authentication から Sourcing への依存を足すので、境界を選ぶ判断手順を適用する。
- principal の名前を `sAMAccountName` にするか、`userPrincipalName` の `@` の前にするか。AD の Kerberos のクライアントは `sAMAccountName` を使う。
- 大文字と小文字の比較。REQ-AUTHENTICATION-042 は完全一致で比べるが、AD の `sAMAccountName` は大文字と小文字を区別しない。

## 計画

1. 未解決の問いと境界の判断を設計に書く。
2. Authentication と Sourcing の仕様を更新してから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いと境界の判断を決める。
- [ ] T002 [Spec] REQ-AUTHENTICATION-042 と REQ-SOURCING-016 を更新する。
- [ ] T003 [App] Connector の写像、対応の保存、Authentication の対応付けを実装する。
- [ ] T004 [Verify] 取り込んだ User の principal で Kerberos のログインが成功し、無効にした User では拒否されることを確かめる。

## 検証

- Kerberos のテスト用の fixture で、設定ファイルに対応を書かずに、取り込んだ User の principal でログインできることを確かめる。

## リスク

- 対応付けを誤ると、別の利用者としてサインインさせる。realm と Connector の結び付けを明示の設定に限り、名前の一致だけで別の realm の User に結び付けない。
