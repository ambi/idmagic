# WI-54453: Split IdManagement rules by operation

作業項目は `wi-54453-split-identity-management-rules-by-operation` である。

WI-54453 は、IdManagement の仕様の規則を、それを置いた操作の義務だけを述べる形に切り直し、題名の型をそろえる。
製品の振る舞い、API、設定は変わらない。

複数の操作の義務を抱えていた規則は、操作ごとの規則に分けた。
新しく加えた規則は次のとおりである。

| 規則 | 内容 |
| --- | --- |
| REQ-IDMANAGEMENT-084 | User の所属グループの参照が返すロール |
| REQ-IDMANAGEMENT-085 | 動的グループへの手動のメンバー操作の拒否 |
| REQ-IDMANAGEMENT-086 | データエクスポートの生成の成功と失敗 |
| REQ-IDMANAGEMENT-087 | データエクスポートのダウンロード |
| REQ-IDMANAGEMENT-088 | データエクスポートの種類とテナントの境界 |

規則の題名は、「〈条件〉の〈対象〉の〈操作〉は、〈結果〉」の一文にそろえた。
題名から作るアンカーが変わったので、規則の見出しへのリンクは張り直す必要がある。
規約は `SPECIFICATION_FORMAT.md` の「操作の節の型」と「規則一件の書式」が定める。
