---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Simplify internal HTTP dependencies and align SCIM layout

HTTP adapter の依存集中を解消し、SCIM を他 context と同じレイアウトへ揃えた。
`support.Deps` を HTTP 横断設定・ライフサイクル (issuer / timeout / emit / health /
shutdown / tenant repo) だけの小さな型へ縮小し、認証・認可の共通依存は
`support.Authenticator`、フェデレーション開始ゲートは `support.ApplicationGate` へ分離した。
各 context の HTTP adapter に専用 `Deps` 型を定義し、`shared/adapters/http/server` の
Register が context ごとの deps を組み立てて各 `RegisterRoutes` へ渡す配線に変更した。
`internal/scim` を直下並置から `adapters/http` / `domain` / `ports` / `usecases` 構造へ
移動し server へ再配線した。公開仕様・route・method・JSON shape・認可挙動・永続化 schema・
SCL はいずれも不変で、純粋な構造整理として全テストが green。`shared/adapters` の移動先名は
`internal/infra` を第一候補として scope に記録済み (実移動は本 work item 対象外)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
