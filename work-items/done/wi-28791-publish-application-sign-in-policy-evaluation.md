---
status: completed
---

# アプリケーションのサインイン方針の評価を Application の公開操作にする

`mise run spec-diff` は規範の差分なしを報告する。
サインインポリシーの評価（`EvaluateSignInPolicy`、`EffectiveSignInRules`、`EffectivePolicyForEvaluation`、`TrustedDeviceAllowedByRules`、`MfaEnrollmentPolicyFromRules`、`UpcomingMfaEnforcementStart`）と判定の型を、公開パッケージの `application/domain` へ移した。`usecases` には別名と、零値の時刻を補う `EvaluateSignInPolicy` を残した。
`domain` は `net` を使えないので、許可 CIDR の照合を `net/netip` へ書き換え、`net` の照合との一致をファズで確かめた。IPv4 射影の接頭辞は IPv4 の接頭辞として照合する。
OAuth2 と Authentication は評価を `application/domain` から呼ぶ。境界の負債の台帳から `private-import` の 2 件を消した。
ポリシーの判定、HTTP の応答、ドメインイベントは変えていない。

全文は `git log -1 -- work-items/active/wi-28791-publish-application-sign-in-policy-evaluation.md` が示すコミットの親、またはそのコミットで読む。
