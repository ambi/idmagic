---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-14
change_kind: bugfix
priority: p1
depends_on:
  - wi-61037-carry-the-original-authentication-into-refreshed-tokens
affected_spec:
  - { path: docs/modules/authentication/features/account-portal.md, requirement: REQ-AUTHENTICATION-039 }
  - { path: docs/modules/authentication/features/account-portal.md, requirement: REQ-AUTHENTICATION-004 }
---

# Bearer で届いた要求のステップアップを RFC 9470 で求める

## 動機

管理ポータルとアカウントポータルは OIDC の SPA であり、API を Bearer のアクセストークンで呼ぶ。
ステップアップの開始と完了は Bearer を付けずにセッションの Cookie で呼ばれ、成立をログインセッションの `step_up_at` に記録する。
一方、元の操作の再送は Bearer で届き、共有の HTTP 支援はアクセストークンの `iat` を認証時刻とみなす。
`StepUpSatisfied` はその値で `max(auth_time, step_up_at)` を評価するので、セッションの記録は判定に届かない。

アクセストークンの有効期間は 10 分で、リフレッシュのたびに `iat` が新しくなる。
リフレッシュから 5 分の間は、再認証を経ずにパスワードの変更、認証の要素の解除、管理画面のステップアップ必須の操作が通ると読める（2026-11-14 のコード読解。T002 で観測する）。
逆に `iat` が 5 分より古い間は、ダイアログで再認証しても再送は拒否され続ける。

ポータルは自分の IdP の OAuth クライアントとして Bearer で API を呼ぶので、リソース側の判断はトークンで完結させる。
RFC 9470（OAuth 2.0 Step Up Authentication Challenge）は、トークンの `auth_time` と `acr` が不足するときに `401` と `insufficient_user_authentication` を返し、クライアントが `max_age` と `acr_values` を付けて再認可する方式を定める。
2026-11-14 に利用者と、これを目標の方式に決めた。

## 対象範囲

- 共有の HTTP 支援が Bearer の要求を認証したとき、アクセストークンの `auth_time`、`amr`、`acr` を認証の結果として渡し、`iat` を認証時刻に使わない。
- ステップアップを求める操作を Bearer で受け、`auth_time` が 5 分より古いか欠けている場合、`401` と `WWW-Authenticate: Bearer error="insufficient_user_authentication", max_age="300"` で拒否する。
- ポータルの SPA は、この応答を受けたら操作の再開に必要な状態を退避し、`max_age` を付けて `/authorize` へ再認可し、戻った後に元の操作を一回だけ再送する。
- 管理発行の API アクセストークンは、保持そのものをステップアップの資格とみなす現在の規則を保つ。
- セッションの Cookie で届く要求は、現在どおり `max(auth_time, step_up_at)` と `403 step_up_required` で判定する。
- ステップアップ必須の全操作で、Bearer の経路の拒否と再認可後の成功を確かめる。

## 対象外

- 制御面の特権操作に MFA の方式を求めること。
  [[wi-468-system-console-privileged-session-assurance]] が、本項目の仕組みに `acr_values` を加えて扱う。
- ポータル以外の OAuth クライアントへ再認可の流れを提供すること。応答は標準どおりなので、各クライアントが対応する。
- ポータルを BFF（サーバー側のセッション）へ移すこと。

## 設計

### 方式の比較

| 候補 | 採否 | 理由 |
| --- | --- | --- |
| RFC 9470：トークンの `auth_time` と `acr` で判定し、不足したら再認可させる | 採用 | リソース側の判断がトークンで完結する。再認可後の `amr` と `acr` を [[wi-468-system-console-privileged-session-assurance]] がそのまま根拠にできる。IdMagic 以外のリソースサーバーにも同じ仕組みが使える |
| Bearer の要求に付くセッションの Cookie を併せて読む | 不採用 | 変更は小さいが、二つの資格情報の整合を自前で保つことになり、同じオリジンの資源にしか効かない |
| アクセストークンに `sid` を載せ、セッションを引く | 不採用 | `sid` はセッションの Cookie の値そのものなので、トークンを受け取るリソースサーバーへ Cookie の値を配ることになる |

### 規範の差分

REQ-AUTHENTICATION-039 に次を加え、ほかのステップアップ必須の要件はこれを上位の要件として参照する。

- アクセストークンで要求を認証したとき、Authentication は、アクセストークンの `auth_time` でステップアップ認証を判定し、`iat` を判定に使わない。
- `auth_time` が 5 分を過ぎたか `auth_time` を持たないアクセストークンでステップアップ認証を求める操作を受けた場合、Authentication は、401 と `WWW-Authenticate` の `error="insufficient_user_authentication"`、`max_age="300"` で拒否し、操作の対象を変えない。

REQ-AUTHENTICATION-004 には、管理発行の API アクセストークンをステップアップ認証済みとして扱う現在の規則を明文化する。

### 実装前に解決する問い

- 再認可の流れ：`max_age=0` で再ログインを強制するか、`max_age=300` で既存のセッションの新しさに任せるか。ログイン直後のセッションは後者で通る。
- 退避する状態：ダイアログの代わりにリダイレクトになるので、フォームの入力をどこまで退避して再送するか。パスワードの変更のように秘密を含む入力は退避しない。
- `StepUpDialog`、ステップアップの開始と完了の API の扱い：ポータルからは使われなくなる。Cookie の経路のために残すか廃止するかを決める。
- 応答の互換性：Bearer の経路では `403 step_up_required` が `401` に変わる。ほかの規範（パスワード、TOTP、WebAuthn、連携、信頼済みデバイス、承認、API トークン、通知設定、アカウント）の拒否の記述をどう参照させるか。

## 計画

1. 実装前の問いを解決し、規範を更新する。
2. T002 で、リフレッシュ直後の Bearer がステップアップ必須の操作を通る現状を観測する。
3. 共有の HTTP 支援、ステップアップの判定、応答、SPA の再認可の順に実装する。

## タスク

- [ ] T001 [Decision] 実装前の問いを解決する。
- [ ] T002 [Acceptance] リフレッシュ直後のアクセストークンでパスワードの変更が受け付けられることを HTTP 境界で観測し、RED にする。
- [ ] T003 [Spec] REQ-AUTHENTICATION-039 と REQ-AUTHENTICATION-004 を更新し、ほかの規範の参照を合わせる。
- [ ] T004 [App] Bearer の認証の結果にトークンの `auth_time`、`amr`、`acr` を渡し、ステップアップの判定と `insufficient_user_authentication` の応答を実装する。
- [ ] T005 [UI] ポータルが `insufficient_user_authentication` を受けたら再認可し、元の操作を再送する。
- [ ] T006 [Verify] ステップアップ必須の全操作（`TestStepUpAnnotatedInterfacesMatchGatedHandlers` が列挙するもの）で、Bearer の経路の拒否と再認可後の成功を確かめる。

## 検証

- 受け入れテスト：リフレッシュ直後の Bearer の拒否、再認可後の成功、Cookie の経路が変わらないこと。
- `mise run test-ui-e2e-file`：ポータルの再認可と再送。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

Bearer の経路のステップアップが画面内のダイアログからリダイレクトになり、入力途中の内容を失う場面が生じる。
退避と再送の範囲を T001 で決め、E2E で確かめる。

セッションを失効させても、発行済みのアクセストークンは最長 600 秒有効である。
これは既存の設計が受け入れている残存リスクであり、本項目は広げない。
