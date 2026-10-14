---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SPECIFICATION.md の Overview に模範例を与え、Standards の 2 列を定義し、Authorization Boundary を Design へ畳む

正準セクションを 7 個から 6 個へ減らした。`Authorization Boundary` を廃止し、節を持っていた 20 文書の内容を `Design` 配下の `### Authorization boundary` へ移した。`SPECIFICATION_FORMAT.md` §3 が既に Design の担当として `security boundaries` を挙げており、2 つの正準セクションが同じ責任を主張している状態だったのを解消した。H3 名は全文書で統一し、`## Design` 直下の最初の H3 に置いたので、Context 横断で認可境界を読む経路は残っている。`SECTION_ORDER` から外したため、以後 `## Authorization Boundary` を書いた文書は `just check-spec` が落とす。
移動と同時に、10 文書から操作ごとのスコープ割り当ての再掲を取り除いた。`x-api-token-scopes` が正本で `just check-admin-scopes` が 210 operation の宣言と `ApiTokenScope` 語彙の対応を双方向に検査しており、散文側は検査されない片割れだった。スコープ語彙と、`provisioning` の変更系判定や `audit` の「対になる変更系スコープは存在しない」のような自明でない結論は残した。この規則自体は `docs/domain/api-tokens/SPECIFICATION.md` の本文に埋まっていたので、`SPECIFICATION_FORMAT.md` §7 へ引き上げた。
`SPECIFICATION_FORMAT.md` に Standards の節 (§5) を新設し、これまでどこにも定義が無かった `Adoption` と `Strength` を独立した 2 軸として定義した。`Adoption` は規格の機能を採るか、`Strength` は採った場合に課す強さである。`optional` × `MUST` が矛盾しないこと、`excluded` が義務を持てないことを明記し、値集合・ID の一意性・正準ヘッダー行・`excluded` の義務禁止を検査に落とした。既存で唯一矛盾していた `NIST63B4-PASSWORD-MINIMUM`（`excluded` × `MUST` で Statement が規格側の視点）を、製品側の宣言へ書き換えた。
Overview は規則ではなく作例で支えることにした。`SPECIFICATION_FORMAT.md` に良い形と悪い形の作例を置き、実在の模範文書 2 本 (`data-keys`、`sourcing`) への参照は `.claude/skills/spec-change/SKILL.md` に置いた。模範を汚していた 5 文（`sourcing`・`identity-governance` の将来計画、`oauth2`・`system`・`identity-management` の読み順の案内）を削除した。
Overview の語句を機械検査する案は試作して破棄した。22 文書で誤検出 0・実違反 2 件を検出できたが、対象語が原理的に閉じず、保守費が検出価値を上回る。閉じた値集合を持つ Standards との違いをこの work item に記録した。Design セクションは方法が無いため対象外とし、その判断根拠（H3 123 個に規約なし、機械判定できるのは見出し言語程度）を Out of Scope に残した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
