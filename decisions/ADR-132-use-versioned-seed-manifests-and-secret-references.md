---
status: accepted
authors: [tn]
created_at: 2026-07-20
---

# ADR-132: seed desired state を versioned YAML manifest と secret reference で外部化する

## コンテキスト

ADR-118 で Seeding bounded context、明示 profile、dry-run、drift policy、冪等 command を導入したが、
desired resource 自体は composition code の Go literal に残った。環境差分や resource 追加のたびに
binary の変更が必要であり、運用設定と DI の責務が再び結合している。単純に YAML へ移すだけでは、
平文秘密値、任意 template、path traversal、曖昧な型変換が新たな攻撃面になる。

## 決定

`models.SeedManifest` を versioned、strictly decoded な YAML desired state とし、
Seeding adapter が domain 型へ変換して既存 contributor へ渡す。root は CLI/startup から明示可能にし、
未指定時は profile ごとの repository default を選ぶ。include は root directory 内のローカル相対 path
だけを深さと総数の上限付きで解決する。YAML merge key、任意 template、remote URL、環境変数展開は
manifest 文法に含めない。

秘密値は `models.SeedSecretReference` だけで表し、manifest に値を置かない。初期 provider は env と
file に絞り、staging/production は file のみ許可する。dry-run も参照の解決可能性を検証するが、
materialized value は plan、log、error に渡さない。file resolver は regular file、size、NUL、
末尾改行の扱いを固定する。

YAML は database fixture とせず、record context の公開 command surface を通す型付き入力とする。
既存の idempotency、manual drift conflict、production profile policy、bounded performance generator は
維持する。performance user は login-disabled とし、既知 password を生成しない。

## 却下した代替案

- Go literal を profile ごとに増やす: 型安全だが運用差分に再ビルドが必要という問題を解消しない。
- YAML に秘密値または `${ENV}` template を埋める: 値と参照の区別が曖昧になり、dump、error、
  dry-run からの漏えいを防ぎにくい。
- 任意 URL include や汎用 template engine: 再現性と監査可能性を落とし、SSRF と template injection の
  攻撃面を seed runner に持ち込む。
- Kubernetes/Vault/cloud provider を初期契約へ含める: provider 固有運用を Seeding core に固定する。
  将来は同じ resolver port の adapter として追加できる。
- DB へ seed manifest と checkpoint を保存する: seed 実行のためだけの新しい状態を増やし、
  ADR-118 の決定的再実行モデルを弱める。

## 影響

- `spec/contexts/seeding.yaml` の `models.SeedManifest`、`models.SeedSecretReference`、
  `models.SeedRequest`、`interfaces.SeedData` と manifest/secret rejection scenarios が規範契約となる。
- `backend/seeding/adapters/manifest` が YAML と filesystem/environment の境界を所有し、
  `backend/seeding/domain` と `backend/seeding/usecases` は parser ライブラリや OS lookup を知らない。
- `seed/manifests` が repository default の運用設定になる。秘密値そのものは置かない。
- provider 追加、remote 配布、prune、GitOps controller は後続の独立判断とする。
