---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で Application → Authentication を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# Application のポリシーを認証事実の値で評価する

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Application → Authentication` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

AuthenticationContext と ACR、AMR への依存を、Application が定める評価入力と境界の変換へ置き換える。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

ポリシーの許可条件、ACR と AMR の意味、HTTP 応答の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[サインインポリシー](../../docs/modules/application/features/sign-in-policy.md)の REQ-APPLICATION-009、010、016 と、`backend/application/domain/sign_in_policy.go`、`internal/usecases/sign_in_policy.go`、`internal/usecases/access_gate.go` を起点にする。

D1、D4、D6 を適用する。
Application はポリシーと評価を引き続き担当し、認証待ち、MFA の成立、実際の第二要素の有無、認証時刻とステップアップ時刻を評価入力にする。
Authentication の ACR と AMR の解釈を Application に複製せず、利用側の変換で既存の認証事実を保つ。
Authentication → Application と、プロトコルからのポリシー評価は残す。

評価を Authentication へ移す案は設定と評価の責任を割り、AuthenticationContext を shared へ移す案は業務の意味を共有へ漏らすため採らない。
単に ACR を文字列に変えて同じ判定を複製する案も採らない。
認証方式を追加する変更では境界の変換を見直し、Application のポリシー規則へ認証方式の列挙を追加しない。

記憶済み端末だけの MFA と実際の第二要素を区別し、認証待ちの拒否、強制開始時刻、再認証の判定とテナントのデフォルトへのフォールバックを保つ。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [ ] T001 [Design] import の全入口と作用を再調査し、境界の型と操作、変換、確定順を設計へ記す。
- [ ] T002 [App] 要求側のポートと所有側の公開操作を結び、逆向きの辺を解消する。
- [ ] T003 [Docs] 公開範囲と現在の設計を同期し、実測で消えた負債だけを削除して依存図を再生成する。
- [ ] T004 [Verify] 既存の保証、正式な配線と境界を検査し、仕様差分を確認する。

## 検証

- `mise run test-go-package -- ./backend/application/domain` と `./backend/application/internal/usecases` のポリシー検査、および `mise run test-go-changed` で全プロトコルとサインインの変換経路を確認する。
  `TestEvaluateSignInPolicyTrustedDeviceSatisfiesMfaOnlyWhenAllowed` と `TestApplicationAccessEvaluatesSignInPolicy` を起点にする。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
