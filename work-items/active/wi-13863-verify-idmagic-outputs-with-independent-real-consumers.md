---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p2
group: development-process
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "IdMagic が発行する SAML の応答とメタデータ、WS-Federation のトークンなどを、IdMagic と独立した実物の消費者に解析させる検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルは変えない。消費者が拒否した出力は、規範への違反なら別の項目に起票する。" }
---

# IdMagic が発行するものを、独立した実物の消費者で検証する

## 動機

IdMagic が発行する SAML の Response と Assertion、IdP メタデータ、WS-Federation のトークンは、テストの中で IdMagic 自身の解析器か、同じ `goxmldsig` と `etree` で読まれている。
発行する側と読む側が同じ思い込みを共有すると、外部の SP が拒否する出力でもテストは通る。
これは、外部の相手の応答を実装者が手で書いていた問題（wi-94480 ほか）の、出力の側の同じ形である。

OAuth と OIDC の出力は、[[wi-33-protocol-conformance-ci]] が OpenID Foundation の適合試験で確かめる予定である。
SAML は wi-33 の対象外であり、SAML の出力を独立した消費者で確かめる項目はない。

## 対象範囲

- IdMagic が発行し外部が解析するものを洗い出し、独立した消費者で確かめるかを出力ごとに決める。候補は SAML の Response と Assertion、IdP メタデータ、SAML の LogoutRequest と LogoutResponse、WS-Federation の RSTR とメタデータ、SET である。
- SAML は、手元で起動できる実物の SP（SimpleSAMLphp または Shibboleth SP）と、`xmlsec1` による署名の検証を候補にし、IdMagic のテスト用の環境から発行した出力を解析させる。
- 検査は、記録の取り込みと同じく手元で実物を動かして出力の受理を確かめる経路と、CI で回す経路のどちらにするかを決める。CI で実物を動かすなら、`test-go-openldap` と同じく既定の検証から分けた区分にする。
- 消費者が拒否した出力を、規範への違反か、消費者の癖かに分ける。

## 対象外

- OAuth と OIDC の適合試験。wi-33 が扱う。
- IdMagic が受け取る要求の記録。[[wi-21740-decide-recordings-for-requests-idmagic-receives]] が扱う。
- Microsoft 365 と Entra ID の実テナントでの受理。[[wi-79-entra-real-tenant-end-to-end-verification]] が扱う。

## 設計

独立した消費者の第一の条件は、IdMagic と XML の正規化と署名の実装を共有しないことである。
`xmlsec1`（libxml2 と xmlsec）は C の実装で、Go の `goxmldsig` と共有しない。
SimpleSAMLphp は PHP の `xmlseclibs`、Shibboleth SP は C++ の Santuario を使うので、どちらも独立している。
一つだけを選ぶなら、署名に加えて Audience、Recipient、時刻、NameID の形式まで SP として判定する Shibboleth SP か SimpleSAMLphp を選ぶ。

消費者の判定を期待値にするのではなく、一次資料が求める判定と消費者の判定が一致することを確かめる。
消費者が一次資料より緩い判定をする場合は、その点を検査から外す。

## 計画

1. 発行する出力と、それを解析する外部の相手を洗い出す。
2. SAML の Response で、実物の SP を一つ動かす経路を作る。
3. 他の出力へ広げるかを、最初の結果を見て決める。

## タスク

- [ ] T001 [Inventory] 発行する出力と消費者を洗い出し、採否を決める。
- [ ] T002 [Harness] 実物の SP または `xmlsec1` に出力を解析させる経路を作る。
- [ ] T003 [Follow-up] 拒否された出力を分類し、違反を起票する。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- 署名を壊した出力を、独立した消費者が拒否する。拒否しなければ経路が空虚である。
- 追加した検査のタスクと、`mise run lint-go`。

## リスク

実物の消費者を動かす検査は遅く、不安定になりやすい。
既定の検証から分け、失敗を消費者の起動の失敗と判定の失敗に分けて報告する。
