---
status: completed
archived_from: 36fa98296ecfad21ec5a718fcb03fc7d9bb51f3b
---

# Application の割り当てが存在しない主体と別テナントの主体を受け入れる

`mise run spec-diff` は、`main` と比較して規範仕様の変更がないと判定した。
`AssignApplication` は、保存前に Application が所有する `SubjectDirectory` ポートを通じて User と Group の主体を解決し、存在しない主体と別テナントの主体を `invalid_request` で拒否する。拒否後には割り当ても `ApplicationAssigned` イベントも残さない。
モジュールアダプターは IdManagement のリポジトリ型をユースケースの外側に保ち、本番構成を通る E2E テストによって `EX-APPLICATION-007-02` を被覆不足台帳から削除した。

全文は `git show 36fa98296ecfad21ec5a718fcb03fc7d9bb51f3b:<このファイルのパス>` で読む。
