---
status: completed
---

# Application のポリシーを認証事実の値で評価する

規範仕様の変更はない。
Application がポリシーの評価に必要な認証事実だけを読む入力契約を定め、Authentication の認証結果が構造的に満たす形へ切り替えた。
ACR と AMR の解釈は Authentication に集約し、認証待ち、MFA、実際の第二要素、認証時刻とステップアップ時刻の値を供給する。
OIDC、SAML、WS-Fed のゲートも同じ入力契約へ揃え、既存テストと期待値を変更せず、信頼済み端末の扱い、強制開始と再認証の判定、デフォルトへのフォールバックを維持した。
Application から Authentication への本番 import を除き、消えた1件の負債だけを削除して、循環負債を31件から30件へ減らした。
ほかの7組の逆向きの辺、ポリシーの許可条件、保存とイベント、HTTP 応答の変更は対象外とした。

全文は `git log -1 -- work-items/active/wi-51457-evaluate-application-policy-from-authentication-facts.md` が示すコミットの親、またはそのコミットで読む。
