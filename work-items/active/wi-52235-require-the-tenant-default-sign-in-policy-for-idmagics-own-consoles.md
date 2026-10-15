---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-16
priority: p2
depends_on: []
change_kind: maintenance
affected_spec:
  - { path: docs/modules/application/sign-in-policy/README.md, requirement: REQ-APPLICATION-016 }
---

# IdMagic 自身の画面へのサインインにテナントのデフォルトのポリシーを適用することを要件にする

## 動機

[REQ-APPLICATION-009](../../docs/modules/application/sign-in-policy/README.md#req-application-009-管理者はアプリケーション別サインインポリシーを設定できる) と [REQ-APPLICATION-010](../../docs/modules/application/sign-in-policy/README.md#req-application-010-管理者はテナントデフォルトサインインポリシーを設定し全アプリに適用できる) は、アプリケーションへのフェデレーションでのポリシーの評価を定める。
一方で、IdMagic 自身の管理コンソールとアカウントポータル（first-party のクライアントと、`return_to` で管理コンソールへ戻る直接のログイン）へのサインインにも、実装はテナントのデフォルトの規則を適用している。
この保証はどの要件にも書かれておらず、守っているテストも要件を引いていない。

[[wi-93464-move-authorization-login-steps-into-authentication]] は、この判定を含むサインインの処理を OAuth2 から Authentication へ移す。
コード発行の直前に評価をやり直す関門は外部のアプリケーションにしかないので、IdMagic 自身の画面はサインインの処理の判定だけが守る。
移動の前に、この保証を要件にし、すべての経路をテストで固定する。

## 対象範囲

- REQ-APPLICATION-016 を追加する。
- 要件の各文を引くテストをそろえる。既存のテストには `//spec:covers` を付け、欠けている経路のテストを足す。

## 対象外

- 振る舞いの変更。今の実装の振る舞いを要件にする。
- WS-Federation の `return_to` でのログイン。WS-Federation の RP はアプリケーションとして評価される。
- サインインの処理の Authentication への移動。wi-93464 で行う。

## 設計

### 要件の差分

[サインインポリシー](../../docs/modules/application/sign-in-policy/README.md)の `## 操作` に「IdMagic 自身の画面へのサインインでの評価」を足し、次の要件を置く。

#### REQ-APPLICATION-016 IdMagic 自身の画面へのサインインにはテナントのデフォルトのポリシーを適用する

- 利用者が IdMagic 自身の管理コンソールかアカウントポータルへサインインするとき、Application は、テナントのデフォルトの規則を評価し、規則を満たすまでサインインを完了させない。
- 認証強度が規則の要求に届かず、利用者が第二要素を登録している場合、Application は、ステップアップ認証を要求し、第二要素が成立するまで管理コンソールへ戻さず、認可コードを発行しない。
- 認証強度が規則の要求に届かず、利用者が第二要素を登録しておらず、MFA の登録も始められない場合、Application は、403 と `access_denied` で拒否する。
- 規則がクライアントのネットワークを許可しない場合、Application は、ステップアップ認証の機会を与えずに 403 と `access_denied` で拒否する。
- `prompt=none` の認可要求がステップアップ認証を必要とする場合、Application は、`login_required` を返し、認可コードを発行しない。
- ステップアップ認証を始めるログインセッションが失効している場合、Application は、401 と `authentication_required` で拒否する。

### 仕様の漏れの分類

| 観点 | 分類 |
| --- | --- |
| 信頼済みデバイスでの省略 | 書かない。[REQ-AUTHENTICATION-026](../../docs/modules/authentication/trusted-device/README.md#req-authentication-026-第二要素の成立時に本人が同意した端末は次回以降の第二要素を省略できる) が経路を問わずに定める |
| MFA の登録の開始 | 書かない。[REQ-AUTHENTICATION-018](../../docs/modules/authentication/mfa/README.md#req-authentication-018-mfa-未登録のユーザーは管理者が承認した登録を終えて同じ認可処理を継続できる) と REQ-AUTHENTICATION-019 が定める |
| 拒否の応答の形 | 要件にする。403 の problem details と 401 は利用者に見える |

### 証拠

| 要件の文 | テスト |
| --- | --- |
| 評価してから完了させる、第二要素の要求 | 追加する `TestFirstPartyBrowserLoginRequiresTheSecondFactorUnderTheTenantDefault`、既存の `TestDirectAdminLoginRequiresTOTPWhenDefaultPolicyRequiresMFA`、追加する `TestAuthorizeResumeRequiresTheSecondFactorUnderTheTenantDefault` |
| 第二要素がない場合の拒否 | 既存の `TestAuthorizeMfaRequiredWithoutSecondFactorIssuesNoCode` |
| ネットワークの拒否 | 既存の `TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403` |
| `prompt=none` | 追加する `TestAuthorizePromptNoneWithStepUpRequiredReturnsLoginRequired` |
| 失効したセッション | 既存の `TestAuthorizeStepUpWithExpiredSessionIssuesNoCode` |

公開状態は `unpublished` である。
足すテストは今の実装で成功するはずであり、成功しない場合は振る舞いの分類へ戻って利用者に確かめる。
各テストが要件を守っていることは、該当する判定を手で壊して落ちることで確かめる。

## タスク

- [x] T001 [Spec] REQ-APPLICATION-016 を追加し、`check-spec` を通す。
- [x] T002 [Test] 既存のテストに `//spec:covers` を付け、欠けている経路のテストを足す。
  既存の 4 件に `//spec:covers` を付け、3 件を足した。足したテストは今の実装で成功した。
- [x] T003 [Verify] 判定を手で壊して各テストが落ちることを確かめ、変更を検証する。
  テナントのデフォルトの評価を呼ぶ 4 か所（ログインの API の first-party、管理コンソールへの直接のログイン、既存のセッションでの `/authorize`、`/authorize/resume`）を一つずつ無効にし、それぞれ REQ-APPLICATION-016 を引くテストが落ちることを確かめた。ログインの API の first-party の経路を検出したのは、足した `TestFirstPartyBrowserLoginRequiresTheSecondFactorUnderTheTenantDefault` だけである。
  `check-spec`、`check-spec-impact`、`lint-go`、`check-work-items` と、`oauth2/handlers_http` と `shared/http/server_http` のテストが成功した。

## 検証

- `mise run check-spec`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run check-spec-impact`

## リスク

- 要件の文が今の振る舞いとずれると、移動の作業がそのずれを「仕様違反」として直そうとする。
  文はすべてテストで裏づけてから確定する。

## 要約

`mise run spec-diff` は REQ-APPLICATION-016 の追加を報告する。
IdMagic 自身の管理コンソールとアカウントポータルへのサインインにテナントのデフォルトのサインインポリシーを適用することを、今の実装の振る舞いのまま要件にした。第二要素の要求、第二要素がない場合とネットワークの拒否、`prompt=none`、失効したセッションの各文を含む。
既存の 4 件のテストに要件を引かせ、テストのなかったログインの API の first-party の経路と、`/authorize/resume` と `prompt=none` のテストを足した。
評価を呼ぶ 4 か所をそれぞれ無効にすると、要件を引くテストが落ちる。
振る舞いは変えていない。
