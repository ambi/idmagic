---
status: completed
---

# 標準が相手に許す省略と追加を、宣言と実装で棚卸しする

`standards.md` を持つ 9 モジュール（oauth2、authentication、provisioning、saml、sourcing、authorization、api-tokens、sharedsignals、ws-federation）について、相手が省略、追加、別の形で送れる要素を一次資料から抜き出し、宣言、本番の入口、省いた形を送る試験と照合した。
規範の差分はない（`spec-diff`：no normative specification change）。

違反は 10 件を bugfix として起票した。
イントロスペクションのヒント（wi-11011）、リフレッシュの `scope` と `resource`（wi-11707）、トークン交換の `audience` と `resource` の宣言の食い違い（wi-18680）、`redirect_uri` の省略（wi-23814）、ループバックのポート（wi-26179、RFC 8252 は未宣言）、`request_uri` の外の重複パラメーター（wi-35790、RFC 9101 は未宣言で EX-OAUTH2-005-04 が拒否を規範にしている）、`Basic` の方式名の大文字と小文字（wi-37685）、`path` のない SCIM の PATCH（wi-38120）、最上位の `sub_id` だけの SET（wi-39876）、`kid` のない上流の ID Token（wi-41129）である。
実装は受け入れるが宣言か試験がないものは、oauth2、saml、ws-federation、authentication ごとに docs の項目にした（wi-41643、wi-63406、wi-77364、wi-82993）。

判断：provisioning は送る側で `Retry-After` の両形式を読み、authorization は外部へ公開する入口がないため起票しない。
SCIM の `path` の値フィルターは wi-248、SSF の送信側は wi-36958、ID-JAG の第三者による確認は wi-38172 が扱う。
DPoP を使う MCP クライアントと利用者から Agent への委譲は、確かめる手段の第三者の実装を特定できず、wi-33 の範囲に残した。
wi-10895 の規則はそのまま使えたため変えていない。要素を拒否せず読まずに捨てる実装は、受け入れの行の `Statement` に結果を書く規則で検出できる。

確認の限界：入口は本番コードを読んで判定し、違反の再現はしていない。各 bugfix の項目が RED で確かめる。
