---
status: completed
archived_from: 077881a1875b77b03e231ee05d48e305196dbeff
---

# 行数または項目の上限を超える CSV のインポートを、一部も適用せずに拒否する

`mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範の差分はない。実装を EX-IDMANAGEMENT-004-02、EX-IDMANAGEMENT-026-02、EX-IDMANAGEMENT-029-02 に合わせた。
User、Group、メンバーシップの CSV のインポートは、`max_rows` と `max_field_bytes` の超過を `max_bytes` と同じく投入の時点で拒否し、400 と `too_many_rows` または `field_too_large` を返す。プレビューのジョブも成果物も作らない。
上限の判定は `CSVReader` と一つの実装で共有し、ドメインの `CheckCSVLimits` と `CopyCSVWithinPolicy` が担う。投入は入力を一度だけ読み、検査しながら成果物ストアへ書く。
三つのジョブは、計画または確定の前に保存したファイル全体を実効の上限で確かめ、超過ならジョブを失敗させる。投入の後に上限が下がった場合も、適用のジョブは先行する行を確定しない。
見出しの誤りと構文の誤りの扱いは変えず、プレビューのジョブの中で一件のエラーとして記録する。

全文は `git show 077881a1875b77b03e231ee05d48e305196dbeff:<このファイルのパス>` で読む。
