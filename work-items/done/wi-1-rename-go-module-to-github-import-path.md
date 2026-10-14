---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Go module path を github.com/ambi/idmagic に変更する

Go module path を `idmagic` から `github.com/ambi/idmagic` に変更した。
リポジトリ内の Go import path を canonical module path に揃え、`justfile` の build ldflags も新しい import path を参照するよう更新した。
ユーザー可視の機能・API 振る舞いは変更していないため、`spec/scl.yaml` は変更していない。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
