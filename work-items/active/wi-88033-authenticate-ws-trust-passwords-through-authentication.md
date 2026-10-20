---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-21
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007, impact: conforms }
---

# WS-Trust の UsernameToken のパスワード認証を Authentication に委ねる

## 動機

パスワードの照合は Authentication の機能だが、WsFederation の能動的な STS（`/trust/usernamemixed`）は、それを `authenticateWsTrustUser` に自前で持つ。
User の `PasswordHash` を直接読み、アカウント単位の試行の制限、番兵のハッシュ、照合、有効性の確認を書き写している。
写しはログイン画面の認証（REQ-AUTHENTICATION-007）と比べて、次を欠く。

| 欠けているもの | 影響 |
| --- | --- |
| IP 単位の試行の制限とレート制限 | 一つの発信元から多数のアカウントを狙う credential stuffing を、ログイン画面の代わりに WS-Trust で行える |
| `AuthenticationFailed` と、閾値を超えた後の `AuthenticationEventAggregated` | WS-Trust でのパスワードの失敗が認証の監査に残らない |
| 失敗の回数のメトリクス | WS-Trust への攻撃が golden signal に現れない |

同じ規則を二箇所に持つため、ログインの防御を強めても WS-Trust は追従しない。
本人の同意の経路が OAuth2 ではなく Authentication に置かれていた件と同じく、機能を所有者でないモジュールに置いた例として、モジュール間の照合で見つけた。

## 対象範囲

- Authentication に、セッションを作らないパスワード認証の公開操作を設ける。入力はテナント、ユーザー名、パスワード、発信元の IP、経路の名前とし、結果は認証した User、または拒否の理由とする。
- ログイン画面と WS-Trust の両方を、この操作へ通す。
- REQ-WSFEDERATION-004 に、IP 単位の試行の制限と認証の失敗の監査を WS-Trust にも適用することを書く。

## 対象外

- WS-Trust の Application の関門（[wi-52958](wi-52958-gate-ws-trust-issuance-by-application-assignment-and-policy.md)）。
- パスワードのハッシュの置き場所の変更。
- WS-Trust での MFA の対応。

## 設計

依存の向きは WsFederation → Authentication で、基準の向き（連携プロトコルが認証を利用する）に合う。
OAuth2 が Authentication の公開操作を使うのと同じ形にし、組み立て地点で結ぶ。
ログイン画面に固有の処理（セッションの作成、必須操作の判定、認可の継続、CSRF）は操作に含めない。
パスワードの期限切れを WS-Trust でどう扱うか（拒否するか、発行を続けるか）は着手時に決め、仕様へ書く。

## 計画

1. 着手時に、ログイン画面のパスワード認証のうち、セッションと画面に依存しない部分を列挙し、操作のシグネチャを設計へ記す。
2. WS-Trust で IP 単位の制限と `AuthenticationFailed` が働かない故障を RED にする。
3. 操作を実装し、ログイン画面の既存のテストを保ったまま WS-Trust を置き換える。

## タスク

- [ ] T001 [Spec] REQ-WSFEDERATION-004 に試行の制限と監査の適用を書く。
- [ ] T002 [Acceptance] WS-Trust の IP 単位の制限と失敗の監査の欠落を RED にする。
- [ ] T003 [App] Authentication の操作を実装し、ログイン画面と WS-Trust を通す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run lint-go`、`mise run check-boundaries`、`mise run check-spec`。
- ログイン画面の既存のテストが変更なしで成功することを確かめる。

## リスク

ログイン画面の失敗の応答（401、429 と `Retry-After`）と WS-Trust の応答（401 と `text/plain`）は異なる。
操作は拒否の理由だけを返し、応答の形は各経路が決める。
