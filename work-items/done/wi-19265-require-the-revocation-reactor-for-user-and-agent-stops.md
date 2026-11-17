---
status: completed
---

# User と Agent を止める操作が、失効の反応器を必須の依存として受け取る

User と Agent を止める IdManagement の操作は、監査の `Emit` と別に、失効の反応器 `userports.AccessRevocationReactor` を必須の依存として受け取る。
停止の 7 種のイベントは `idmusecases.EmitRevoking` だけで発行し、反応器が未配線なら監査を記録してから `ErrAccessRevocationUnavailable` で失敗する。
`Module.UserLifecycleCommands`、`Module.AdminUsers`、管理 API の経路は nil の反応器を構築時に拒否する。
`deps_http.Deps.ReactiveEmit`、`bootstrap.Dependencies.ReactiveEmit`、routes と testing_stack の即席の合成を消し、期限 5 秒の反応を一つにした。
`spec-diff` は規範の差分なし。Authentication と OAuth2 のイベントを失効へ変換する `userRevocationEmit` は失敗の扱いが異なるので対象外とした。
テストの組み立ては、何もしない `ignoreRevocation` を各テストパッケージで明示して渡す。
