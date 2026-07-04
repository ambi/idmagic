---
id: idp-wi-71-application-sign-on-policy
title: "アプリケーション単位のサインオンポリシー (条件付きアクセス / step-up) を導入する"
created_at: 2026-06-27
authors: ["tn"]
status: pending
risk: high
---

# Motivation
Okta の App Sign-On Policy も Entra ID の Conditional Access も、認証強度や許可条件を
テナント全体ではなくアプリケーション単位で要求できる。「この業務アプリは MFA 必須」
「社外ネットワークからはアクセス不可」「管理アプリは passkey 必須」のように、
リスクの高いアプリに強い要件を課し、低リスクなアプリは摩擦を減らす。

idmagic は現状、認証強度をアプリ単位で要求する仕組みを持たない。
[[wi-43-account-portal-step-up-auth]] で step-up 認証と AuthenticationContext (acr/amr)
の基盤はあるが、フェデレーション開始時に「このアプリが要求する強度を満たしているか」を
判定して不足なら step-up へ誘導する経路がない。本 WI は [[wi-69-application-catalog-aggregate-and-assignment]]
の Application に sign-on policy をぶら下げ、protocol を問わず (OIDC / SAML / WS-Fed)
共通に評価する。割当ゲート (wi-69) の次段として、満たさない要求は fail-closed で拒否し、
step-up 可能なら昇格を促す。

# Scope
- **decision**:
  - 新規 ADR-067: アプリ別サインオンポリシーの所有と評価点を確定する。ポリシーは ApplicationCatalog が所有し、評価は各 protocol context のフェデレーション開始時に Authentication の AuthenticationContext を入力として行う。条件 (要求 acr / 要求 factor、ネットワーク、デバイス信頼、再認証 max_age) の初期サポート範囲、 満たさない場合の挙動 (step-up 誘導 or 拒否) を決める。
- **scl**:
  - ApplicationCatalog に AppSignOnPolicy / SignOnRule / AccessCondition (network / device / reauthMaxAge) / RequiredAuthnLevel (required acr / factor) を追加する。
  - interface: 管理者の sign-on policy CRUD (Application 配下)。評価は既存 フェデレーション開始 interface (Authorize / WsFederationSignIn / 将来 SAML SSO) 内で行う。
  - event: AppSignOnPolicyUpdated / AppAccessDeniedByPolicy / AppStepUpRequired。
  - invariant: AppPolicyFailClosed (要求を満たせない場合トークン/アサーションを発行しない)、 AppPolicyEvaluatedAcrossProtocols (全 binding で同じポリシーを評価する)。
  - permission: AdminApplicationPoliciesManage。
- **go**:
  - sign-on policy の persistence (application_sign_on_policies テーブル、tenant scope)。
  - フェデレーション開始経路でポリシー評価器を呼び、AuthenticationContext と条件を突き合わせ、 不足時は step-up へ誘導するか fail-closed で拒否する。
- **http**:
  - /admin/applications/{id}/sign-on-policy の取得/更新。
- **ui**:
  - admin: Application 詳細に sign-on policy エディタ (要求強度・条件)。
  - account/auth-flow: ポリシー不足時の step-up 画面誘導と拒否時の明確な理由表示。

# Out of Scope
- リスクスコアリング/UEBA など動的リスク評価エンジン。初期は静的条件のみ。
- デバイス信頼の実証明 (MDM/attestation) 本体。条件の入力点だけ用意し実体は将来 WI。
- アプリ単位のセルフサービス申請/承認 (別途検討)。
- 割当そのもの ([[wi-69-application-catalog-aggregate-and-assignment]])。

# Verification
- `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `bun run yaml-check:work-items` (in: tools)
- `bun run yaml-check:scl` (in: tools)
- 手動: MFA 必須ポリシーを設定したアプリに単要素セッションでアクセスし step-up に 誘導され、昇格後に SSO 完了することを確認する。step-up 不可条件では拒否されることを確認する。

# Risk Notes
認証強度の判定を全フェデレーション開始経路に漏れなく適用する必要があり、1 経路でも
評価が抜けると「ポリシーを迂回してログインできる」欠陥になる (wi-69 の割当ゲートと同じ
クラスのリスク)。ADR-067 で評価点と fail-closed を先に固定し、AuthenticationContext を
唯一の入力にして protocol 横断で同じ判定器を通す。
