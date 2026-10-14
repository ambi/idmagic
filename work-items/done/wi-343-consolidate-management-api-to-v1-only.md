---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 管理 API・自己管理 API のパスを `/v1/` のみに一本化する (未リリース時の一括移行)

[[wi-297-management-api-versioning-and-deprecation-policy]] が導入した「無版パス = v1 の
エイリアス」設計を、未リリースであることを理由に「`/v1/` のみ・無版パスは存在しない」へ
一本化した。対象は `/api/admin/*` (201 interface) と `/api/account/*` (31 interface) の
bindings path のみで、`/api/branding`・`/api/auth/*`・protocol endpoint・ops endpoint は
ADR-156 の版管理スコープ外のまま変更していない。
SCL (232 binding)・Go (route 登録 16 ファイル + 参照テスト 44 ファイル)・frontend (呼び出し
83 ファイル 332 箇所) を機械的に書き換え、`backend/shared/http/support_http/versioning.go`
(`RegisterVersionAliases` エイリアス機構) と `tools/scl-to-openapi` のエイリアス複製ロジックを
削除した (現行版が最初から `/v1/` を含むため、実行時・生成時どちらのエイリアス機構も不要になった)。
機械的書き換えは 2 種類の見落としを生んだが、どちらも既存の安全網 (Go: 型検査でなく
test suite の red、UI: `typecheck-ui` の型エラー) で即座に検出できた:
1. `/realms/xxx/api/admin/...` のように prefix が付く Go の文字列 literal
(最初の正規表現が文字列の先頭一致だけを見ていたため見逃した)。
2. frontend の相対 import 文 `from '../../api/admin'` (HTTP パス文字列と同じ
`/api/admin` という部分文字列を含むが別物)。
`spec/idmagic.openapi.baseline.json` はこの WI 完了時点のスナップショットで無条件に
上書きした (これは未リリース前提の一度限りの操作であり、リリース後は行わない)。
ADR-156 は同一番号のまま本文を更新した (supersede はしない。まだリリースされていない
決定の即時修正のため)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
