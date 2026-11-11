---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-12
priority: p3
depends_on: [wi-95114-remove-entra-federation-one-domain-at-a-time]
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
---

# Entra のドメインフェデレーションの構成、確認、解除を Microsoft Graph で自動化する

## 動機

いまの Entra のドメインフェデレーションは手動の方式である。
管理者は IdMagic が返す Graph PowerShell の手順で Entra を変え、読み戻した登録値の観測文書を IdMagic に取り込む（[[wi-96973-guide-entra-federation-setup-and-verify-applied-settings]]、[[wi-95114-remove-entra-federation-one-domain-at-a-time]]）。
Okta は、管理者が Microsoft にサインインして権限に同意すると、Okta 自身が Graph でドメインをフェデレーションし、アプリを外すとフェデレーションも外す自動の方式を推奨している（[Okta: Configure WS-Federation for Office 365](https://help.okta.com/en-us/Content/topics/apps/apps_configuring_ws-federation.htm)）。
自動の方式なら、PowerShell の実行、観測文書の取り込み、手作業の転記の誤りがなくなる。
ただし IdMagic にはまだ Graph のクライアントがなく、この機能のためだけに作ると、Entra のテナントへの接続、同意、トークンの扱い、強い書き込みの権限を一度に抱え込む。

## 着手の条件

Entra ID との同期（Provisioning による利用者の作成など）で、IdMagic が Graph のクライアントを持つ作業が生じたときに着手する。
その接続、同意、トークンの扱いを共有できるかを、その作業の設計と合わせて判断する。
条件が満たされるまでは手動の方式を維持する。

## 対象範囲

- 管理者の同意による Entra のテナントへの接続と、ドメインの構成、更新、読み戻し、Managed への切り替えまたは以前の構成の復元を IdMagic が Graph で行う。
- 自動で読み戻した登録値を、観測記録と同じ照合と解除の条件に使う。
- 手動の方式との関係（置き換える、併存させる、テナントごとに選ぶ）を決める。

## 対象外

- Graph のクライアントそのものと、Entra ID への利用者の同期。着手の条件となる別の作業項目が扱う。
- Entra Connect の構成と、パスワードハッシュ同期やパススルー認証の構築。

## 設計

着手時に次の問いを解決する。

- 資格情報を長期間保存せず、操作のたびに管理者の委任の同意を得る形にできるか。保存する場合の暗号化、失効、監査。
- 必要な権限（`Domain.ReadWrite.All`、`Domain-InternalFederation.ReadWrite.All` など）の最小化と、ドメインの認証方式を書き換えられる権限を IdMagic が持つことの脅威モデルへの影響。
- wi-96973 が採らなかった「全自動方式は資格情報の保管と外部への書き込みを増やす」という判断を、着手の条件となる作業の後にも維持するか。
- 外部の書き込みとローカルの保存が部分的に失敗したときの再開。wi-95114 の「外部の切り戻しを確かめてからローカルを外す」順は維持する。
- 手動で取り込んだ観測記録と、Graph から IdMagic が直接読んだ観測の区別。

## 計画

1. 着手の条件となる作業の Graph の接続の設計を読み、共有の可否を決める。
2. 要件と TypeSpec を更新し、手動の方式との関係を決める。
3. 自動の構成、確認、解除を実装し、検証用テナントで確かめる。

## タスク

- [ ] T001 [Research] 共有できる Graph の接続と、同意とトークンの扱いを決める。
- [ ] T002 [Spec] 自動の構成、確認、解除の要件と、手動の方式との関係を定める。
- [ ] T003 [App] Graph による構成、読み戻し、切り戻しを実装する。
- [ ] T004 [Verify] 検証用テナントで、構成、解除、部分失敗からの再開を確かめる。

## 検証

- 着手時に、変更した要件と経路に合わせて選ぶ。
- `mise run check-work-items`：起票した記録の形式と参照を確かめる。

## リスク

IdMagic が Entra のドメインの認証方式を書き換える権限を持つと、IdMagic の侵害がテナントの全利用者のサインインの乗っ取りにつながる。
権限の範囲、保持期間、監査を着手時に脅威モデルへ記録する。
