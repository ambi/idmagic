---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# アプリケーションのサインオンポリシーをサインインポリシーとして再設計する

「サインオンポリシー」を「サインインポリシー」へ改称し、内部表現 (SCL entity / interface / DTO 名、
HTTP パス `/sign-in-policy`、DB テーブル `application_sign_in_policies`、監査イベント `AppSignInPolicyUpdated`、
Go / TS 型、UI 文言) まで `SignOn` → `SignIn` に統一した。要求認証強度は自由入力 acr / factor を廃止し、
制約 enum `RequiredAuthnStrength` (Password / Mfa) を内部 acr URN へ写像する形にした。static condition は
実評価できる構造化条件のみとし、network を許可 CIDR リスト (`network_allow_cidrs`) に、device を撤去した。
評価器は fail-closed で、MFA 不足は step-up 誘導 (OIDC) / 拒否 (SAML・WS-Fed)、CIDR 不一致・クライアント IP
不明は拒否する。クライアント IP を OIDC / SAML / WS-Fed の 3 経路で評価器に配線した (`Deps.ClientIP`、
TRUSTED_FORWARDED_HOPS ベース)。CIDR は保存時に検証する。PostgreSQL レベルでは未リリースのため、
旧テーブル改称と rule JSON 変換用の個別 migration SQL は残さず、`deploy/schema/postgres.sql` の
現在形に統合した。評価点と所有関係は ADR-079 を更新して記録した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
