---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p1
depends_on: [wi-10895-declare-what-standards-let-peers-omit]
change_kind: maintenance
spec_impact: { kind: none, reason: "既存の宣言と実装を調べて欠陥を起票する作業であり、規範の行の追加と実装の修正は起票した各項目が行う。" }
---

# 標準が相手に許す省略と追加を、宣言と実装で棚卸しする

## 動機

[[wi-62545-accept-client-assertions-without-client-id]] の欠陥は、標準が OPTIONAL とする要素を IdMagic が必須にし、規範の文書も試験もそれを検出できなかったものである。
原因は、標準の宣言が検査の規則に偏り、土台の標準が宣言されず、試験の要求を自分で組み立てていたことにある。
同じ条件は他の標準と入口にも残り得る。
[[wi-10895-declare-what-standards-let-peers-omit]] が足す規則で、既存の宣言と実装を一度に調べる。

## 対象範囲

次の条件が重なる箇所を調べ、違反を見つけたら一件ずつ bugfix の作業項目として起票する。

| 調べる対象 | 例 |
| --- | --- |
| 標準が OPTIONAL とする要素を、製品が必須にするか特定の値を前提にする入口 | トークン、PAR、イントロスペクション、失効、認可、デバイス認可、CIBA の各エンドポイントの任意のパラメーター。`private_key_jwt` で `client_id` を省いた要求は、`/token` 以外で試験されていない |
| 機能を 1 行に要約した宣言の行 | `docs/modules/oauth2/standards.md` の 93 行のうち、検証の項目だけを並べた行 |
| 宣言されていない土台の標準 | RFC 7521 と RFC 7523 のように、採用した標準が依存する枠組み |
| 試験の要求を自作し、第三者の実装で確かめていない経路 | DPoP を使う MCP クライアント、利用者から Agent への委譲、ID-JAG |

対象のモジュールは、`standards.md` を持つ全てのモジュール（oauth2、authentication、provisioning、sourcing、authorization、api-tokens、sharedsignals、ws-federation）である。

## 対象外

- 見つけた違反の修正。起票した各項目が仕様と実装を直す。
- 外部の適合性試験の導入。[[wi-33-protocol-conformance-ci]] が扱う。
- ID-JAG の外部 IdP との相互運用。[[wi-38172-verify-id-jag-interoperability-with-an-external-idp]] が扱う。

## 設計

標準ごとに、相手が省略できる要素、追加できる要素、複数の形を取れる要素を一次資料から抜き出す。
抜き出した各要素について、宣言の行、実装の入口、その要素を省いたか足した要求を送る試験の三つを対応付ける。
三つのどれかが欠けたものを、欠けた種類ごとに分類する。

| 欠けたもの | 扱い |
| --- | --- |
| 実装が拒否する | 違反。bugfix を起票する |
| 実装は受け入れるが、宣言か試験がない | 規範の取りこぼし。宣言の行と試験を足す作業項目を起票する。モジュールごとに一件にまとめてよい |
| 製品が採用しない | 宣言の `excluded` の行で明示されているかだけを確かめる |

棚卸しの表は、この記録の設計に残してから起票する。

## 計画

1. oauth2 から始め、規則が実際の棚卸しに使えるかを確かめる。使いにくければ [[wi-10895-declare-what-standards-let-peers-omit]] の規則を直す。
2. 残りのモジュールを調べる。
3. 見つけたものを起票する。

## タスク

- [ ] T001 [Survey] oauth2 の標準について、相手が省略または追加できる要素と、宣言、入口、試験の対応を表にする。
- [ ] T002 [Survey] 残りのモジュールについて、同じ表を作る。
- [ ] T003 [Docs] 違反と規範の取りこぼしを、作業項目として起票する。
- [ ] T004 [Verify] `mise run check-work-items` と `mise run check-links` を通す。

## 検証

- `mise run check-work-items`：起票した項目の形式と依存を確かめるため。
- `mise run check-links`：棚卸しの表から一次資料と項目へのリンクを確かめるため。

## リスク

一次資料からの抜き出しが漏れると、同じ種類の欠陥が残る。
各標準の抜き出しには、その標準の「OPTIONAL」「MAY」と、要素の省略時の扱いを述べる節を必ず読む。
