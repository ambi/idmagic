---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 機能の成熟度と変更種別から必要な利用者向け文書を検査する

`mise run spec-diff` は `no normative specification change against main` を返した。製品の規範仕様は変更せず、変更種別、TypeSpec と OpenAPI の差分、`FeatureRegistry` の成熟度差分から利用者向け文書の最低影響を導出する純粋検査を追加した。作業空間アダプターは Git と生成 OpenAPI の読取りを担い、完了時にリリース文書の実在、作業項目 ID、安定した仕様参照、成熟度昇格の証拠を検査する。変更履歴と更新ガイドは現在状態を複製せず、リリース固有の差分だけを所有する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
