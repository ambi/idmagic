---
id: idp-wi-114-application-login-policy-language-and-conditions
title: "アプリケーションのサインオンポリシーをサインインポリシーとして再設計する"
created_at: 2026-07-04
authors: [tn]
status: completed
risk: high
---

# Motivation
現在の「サインオンポリシー」は、他の画面で使っている「ログイン / サインイン」語彙とずれており、管理者に余計な理解負荷を与えている。
"sign-on policy" は Okta / Ping レガシー寄りの語で業界の普遍語ではなく、内部識別子として温存する積極的な理由も薄い。
さらに ACR、factor、Password、MFA、再認証最大経過秒数、ネットワーク条件、デバイス条件が UI 上で何を意味するのか分かりづらい。
network / device に任意文字列を入力できる形は、実際に評価できる条件なのか、将来の入力点なのかが曖昧で、ポリシーがあるように見えて実効性が不明になる（現状 network / device は評価器が常に fail-closed 拒否する見せかけの入力欄）。
管理者が自然な語彙と制約された選択肢で、実際に評価されるサインイン要件を設定できるようにする。

# Decisions
- **命名**: 「サインインポリシー」に改称し、**内部表現まで含めて完全改称する**。
  UI 文言・SCL description に加え、SCL entity / interface / DTO 名、HTTP パス、DB テーブル、監査イベント名も `SignOn` → `SignIn` に統一する。
  （`AppSignOnPolicy`→`AppSignInPolicy`、`SignOnRule`→`SignInRule`、`/sign-on-policy`→`/sign-in-policy`、`application_sign_on_policies`→`application_sign_in_policies`、`AppSignOnPolicyUpdated`→`AppSignInPolicyUpdated` 等）。
- **認証強度**: 自由入力の acr / factor を廃止し、制約 enum `RequiredAuthnStrength`（`Password` = 追加要求なし / `Mfa` = 第二要素必須）に写像する。内部の acr URN（`urn:idmagic:acr:pwd` / `urn:idmagic:acr:mfa`）と amr へ 1:1 で対応させる。
- **再認証**: `reauth_max_age_seconds` を維持し、認証または step-up の recency として評価する。
- **network**: 自由入力を廃止し、実際に評価できる構造化条件「許可 CIDR リスト（`network_allow_cidrs`）」に限定する。クライアント IP が許可 CIDR に含まれなければ fail-closed で拒否する。IP を取得できない場合も拒否する。
- **device**: 撤去する（将来 WI）。

# Scope
- `spec/contexts/application.yaml`
  - `AppSignInPolicy` / `SignInRule` / `RequiredAuthnLevel` / `AccessCondition` の用語・型を見直す。UI と API の表示名・識別子を「サインインポリシー」へ統一する。
  - ACR / factor をそのまま入力させる契約を廃止し、`RequiredAuthnStrength`（Password / Mfa）に写像する value object / enum を定義する。
  - Password / MFA / step-up / 再認証 max age の意味を SCL 上で明確にし、既存 AuthenticationContext（acr / amr）との対応を記述する。
  - network 条件を `network_allow_cidrs`（構造化）に限定し、device 条件は撤去する。
- ADR
  - ADR-079 を更新し、サインインポリシーの語彙、管理者向け選択肢（Password / Mfa）、内部 ACR / AMR への写像、network CIDR の評価と fail-closed、device 撤去を記録する。
- Go / HTTP
  - `SignOn`→`SignIn` の完全改称（型 / interface / DTO / パス / DB テーブル / イベント）。既存データは新テーブルへ移行する。
  - 評価器を制約モデルに合わせて更新し、構造化条件（MFA 必須 / 再認証 max age / 許可 CIDR）だけを評価する。CIDR はクライアント IP を突き合わせて fail-closed に評価する。3 プロトコル開始経路（OIDC / SAML / WS-Fed）へクライアント IP を配線する。
  - CIDR の妥当性は保存時に検証する。
- UI
  - 管理者向け表示文言を ACR / factor などの内部語彙から切り離し、「パスワードのみ」「MFA 必須」「再認証を要求」など理解できる表現にする。
  - ネットワーク条件は自由入力ではなく許可 CIDR リストの入力 UI にする。device 入力欄は撤去する。

# Out of Scope
- 全アプリケーションにまたがる既定サインインポリシーの導入。
- リスクスコアリング、UEBA、MDM / デバイス証明などの高度な条件評価エンジン（device 条件の実体）。
- 新しい認証要素そのものの追加。

# Verification
- `just yaml-check-scl`
- `just verify-go`
- `just verify-ui`
- 手動: 既存ポリシーを読み込み、管理者向け語彙で同じ意味に表示・保存できること。
- 手動: 未対応の自由入力（旧 network / device 文字列）を保存できないこと。
- 手動: MFA 必須、再認証要求、許可 CIDR 不一致時の拒否または step-up が既存の federation 開始経路（OIDC / SAML / WS-Fed）で一貫して評価されること。

# Risk Notes
ポリシー語彙を変えるだけに見えて、内部の ACR / AMR 判定や fail-closed 評価に直結するためリスクは高い。
完全改称は DB テーブル・監査イベント名・API パス・型の広範囲に及ぶため、どれかがずれると管理者が意図した強度と実際の評価が食い違う。
network CIDR 評価は 3 プロトコル経路すべてでクライアント IP を漏れなく配線する必要があり、1 経路でも抜けると「IP 制限を迂回できる」欠陥になる。IP 不明時は fail-closed で拒否する。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  「サインオンポリシー」を「サインインポリシー」へ改称し、内部表現 (SCL entity / interface / DTO 名、
  HTTP パス `/sign-in-policy`、DB テーブル `application_sign_in_policies`、監査イベント `AppSignInPolicyUpdated`、
  Go / TS 型、UI 文言) まで `SignOn` → `SignIn` に統一した。要求認証強度は自由入力 acr / factor を廃止し、
  制約 enum `RequiredAuthnStrength` (Password / Mfa) を内部 acr URN へ写像する形にした。static condition は
  実評価できる構造化条件のみとし、network を許可 CIDR リスト (`network_allow_cidrs`) に、device を撤去した。
  評価器は fail-closed で、MFA 不足は step-up 誘導 (OIDC) / 拒否 (SAML・WS-Fed)、CIDR 不一致・クライアント IP
  不明は拒否する。クライアント IP を OIDC / SAML / WS-Fed の 3 経路で評価器に配線した (`Deps.ClientIP`、
  TRUSTED_FORWARDED_HOPS ベース)。CIDR は保存時に検証する。PostgreSQL レベルでは未リリースのため、
  旧テーブル改称と rule JSON 変換用の個別 migration SQL は残さず、`deploy/schema/postgres.sql` の
  現在形に統合した。評価点と所有関係は idp-ADR-079 を更新して記録した。
- **Verification Results**:
  - `just yaml-check`
    - result: ok (SCL 12 files, work-items, ids all OK)
  - `just lint-go`
    - result: ok (`golangci-lint run ./...`: 0 issues)
  - `just verify-go`
    - result: ok (`go test -race ./...`: ok)
  - `just verify-ui`
    - result: ok (format check, lint, typecheck, build)
  - 手動 (deploy): PostgreSQL レベルでは未リリースのため、個別 migration SQL は不要。
    本番/ステージング反映時は `deploy/schema/postgres.sql` の dry-run / apply を確認する。
