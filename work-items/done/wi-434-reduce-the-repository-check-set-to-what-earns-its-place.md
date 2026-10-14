---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 検査の数を、置いている理由を言えるものだけに戻す

`mise run spec-diff` は空である。意味上の差は、[[wi-433-repository-toolchain-covers-every-owned-source]] が入れた 7 つの検査のうち、置いている理由を言えなかった 4 つ (shfmt、actionlint、hadolint、k6 の Biome) を外し、mise の固定バージョンが 6 個から 3 個に減ったことである。`load/biome.json` と `format-repo` は不要になったので削除した。shellcheck は誤検出 2 件を名指しで外し、`dev.sh` の抑制コメントを消した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
