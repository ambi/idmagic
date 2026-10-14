---
status: cancelled
archived_from: 557cf65fa80aeac032a59e0ac41a4fde2e787622
---

# CSV user import を IdManagement から inbound の適所へ移設する

未実装のまま cancelled。本 WI は ADR-128 §影響 の「CSV import は適所でない、inbound へ移設すべき」
という申し送りを前提としていたが、[[wi-258-inbound-integration-taxonomy]] の完了時点でこの前提は
ユーザー自身の判断により覆っていた (wi-258 Human Decisions: 「管理者 CSV import の帰属
(IdManagement に残す)」)。この判断は [[ADR-141-inbound-identity-sourcing-taxonomy]] 決定 5 に
「管理者 CSV import は IdManagement に残す (ADR-128 §影響 の申し送りを覆す)」として記録済みだった
が、ADR-141 は本 WI 着手時点で `status: suggested` のまま残っており、wi-258 Residual Risk が
「受理時に wi-260 の廃止 (または将来の scheduled file feed 限定への書き換え) が必要」と明記して
いた。wi-259 (SCIM の `backend/sourcing/scim` 移設) は既に ADR-141 の構造を確定事実として実装・
完了しており (2026-07-25)、ADR-141 の frontmatter 更新だけが取り残されていた。
本 WI の着手にあたり、この矛盾をユーザーへ提示し裁定を仰いだ結果、ADR-141 の内容を追認し
(status を `suggested` → `accepted` に更新)、その決定 5 に従って本 WI を cancelled とした。
CSV user import (`backend/idmanagement/usecases/user_import.go` /
`admin_user_import_handler.go`) は IdManagement に residing のまま、コード移動は行っていない。
将来 scheduled file feed (HR システムの nightly SFTP CSV 等、ADR-141 決定 5) を実装する場合は、
それを受ける slim な後継 WI をその時点で新規に立てる (本 WI の再利用はしない — 対象が
「管理者の一回性 upload」と「source binding を持つ scheduled feed」で別物のため)。

全文は `git show 557cf65fa80aeac032a59e0ac41a4fde2e787622:<このファイルのパス>` で読む。
