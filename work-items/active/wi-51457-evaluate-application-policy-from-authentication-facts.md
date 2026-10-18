---
status: in_progress
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

### 着手時に確定した入力境界

本番の逆向きの import は Application の `domain/sign_in_policy.go`、`internal/usecases/sign_in_policy.go`、`internal/usecases/access_gate.go` の3か所である。
認証の文脈から読む値は、認証待ち、MFA の成立、記憶済み端末以外の第二要素の成立、認証時刻、ステップアップ時刻の5つに限られる。
評価器は永続化とイベント発行を行わず、ゲートは従来どおりアプリの検索、状態、所属と割り当て、個別ポリシー、デフォルトの順に読み取る。
拒否後のプロトコルのイベントと発行の停止、認証開始の編成、SQL とトランザクションは変更しない。

D1、D4、D6 に従い、Application の `domain.SignInAuthentication` を要求側の読み取り専用の入力契約にする。
操作は `Pending() bool`、`MFAAuthenticated() bool`、`SecondFactorAuthenticated() bool`、`AuthenticatedAt() int64`、`SteppedUpAt() int64` とし、すべて入力だけで決まる認証事実を返す。
`Pending` は認証結果が存在しない場合も true を返し、未取得と認証待ちの共通の拒否を維持する。
AuthenticationContext は既存の認証結果をこの契約へ構造的に適合させる。
MFA は既存の `ACRSatisfies(ACR, ACRMFA)` へ委ね、実際の第二要素は `AMRTrustedDevice` を除いた `IsMfaAMR` の成立で表す。
Application は両方の事実とルールの信頼済み端末の許可を組み合わせ、認証方式の文字列を知らない。

具体的な入力構造体と各入口での変換を置く案も比較した。
Authentication の文脈には既に読み取りのメソッドがあり、正式な評価の呼び出しはすべてその文脈を持つため、同じ値を各入口で組み立て直すより、認証結果自身から事実を読む契約を採る。
既存の文脈を引き続き値の供給元とし、旧型のための互換評価器と変換専用の組み立てパッケージは追加しない。
Application 自身の評価を Authentication へ移す案は D1 に反し、ACR と AMR を文字列で公開し直す案は認証方式の判断を複製するため採らない。
認証方式を増やすときは Authentication の語彙と事実への射影だけが変わり、Application とプロトコルには列挙を追加しない。

OIDC、SAML、WS-Fed の HTTP 側の ApplicationGate は同じ入力契約を受ける。
SAML と WS-Fed の既存のゲートアダプターは、プロトコルの認証文脈をそのまま構造的に渡す。
必須の配線を増減せず、既存のインターフェースと実装の一致は正式な組み立てのコンパイルで確認する。
nil の文脈がインターフェースへ入る場合も拒否できるよう、AuthenticationContext の `Pending` を nil に対して true とする。
非 nil の入力に関する既存の意味は変えない。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [x] T001 [Design] import の全入口と作用を再調査し、読み取りの入力契約、認証事実への射影、確定順と比較基準を設計へ記した。
- [x] T002 [App] 要求側の入力契約へ AuthenticationContext を適合させ、既存の78テストを変更せずにポリシーとゲートの成功を確認した。境界検査で逆向きの辺の解消を確認した。
- [x] T003 [Docs] 公開範囲と現在の設計を同期し、実測で消えた1件だけを負債台帳から削除して依存図を再生成した。
- [x] T004 [Verify] 変更していない既存テストで評価、認証と全プロトコルの逆依存を race 検査し、正式な配線、境界、Go の静的検査、文書と仕様差分を確認した。

## 検証

- `mise run test-go-package -- ./backend/application/domain` と `./backend/application/internal/usecases` のポリシー検査、および `mise run test-go-changed` で全プロトコルとサインインの変換経路を確認する。
  `TestEvaluateSignInPolicyTrustedDeviceSatisfiesMfaOnlyWhenAllowed` と `TestApplicationAccessEvaluatesSignInPolicy` を起点にする。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

今回の比較基準は `e3435021d` とする。
既存のテストファイルと期待値は変更せず、`test-go-package` で Application の評価とゲート、Authentication の認証事実を局所検査する。
`test-go-changed -- e3435021d` で認証、OIDC、SAML、WS-Fed、正式な組み立ての逆依存を race 付きで検査する。
HTTP の待受けを含む検査はサンドボックス外で実行する。
強制開始は REQ-AUTHENTICATION-019、信頼済み端末と第二要素の区別は REQ-AUTHENTICATION-027 の既存テストが固定する。
REQ-APPLICATION-009、010、016 の正式な HTTP の経路、拒否時の無作用とイベントも変更しない既存テストで確認する。
テストを変えない構造変更なので、独立した RED と変更耐性の検査は追加しない。
新しい文字列解析と正規化を導入せず、ACR と AMR の既存の解釈を移すため、ファズ対象も追加しない。
最終検査は `lint-go`、`check`、`render-docs`、`check-published-api-compat` と上述の race 検査を選ぶ。
UI と公開 HTTP 契約を変更しないため、ブラウザー検査は追加しない。
既存の `FuzzClientIPAllowed` は保存時に拒否される不正な CIDR の2シードを対象外にするため、そのスキップと親の区分を成功へ数えない。
今回変更しないネットワークの解析の範囲であり、認証事実の入力の証拠とは区別する。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。

## 要約

規範仕様の変更はない。
Application がポリシーの評価に必要な認証事実だけを読む入力契約を定め、Authentication の認証結果が構造的に満たす形へ切り替えた。
ACR と AMR の解釈は Authentication に集約し、認証待ち、MFA、実際の第二要素、認証時刻とステップアップ時刻の値を供給する。
OIDC、SAML、WS-Fed のゲートも同じ入力契約へ揃え、既存テストと期待値を変更せず、信頼済み端末の扱い、強制開始と再認証の判定、デフォルトへのフォールバックを維持した。
Application から Authentication への本番 import を除き、消えた1件の負債だけを削除して、循環負債を31件から30件へ減らした。
ほかの7組の逆向きの辺、ポリシーの許可条件、保存とイベント、HTTP 応答の変更は対象外とした。
