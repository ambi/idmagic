---
name: scl-render
description: Regenerate SCL-derived artifacts (HTML views, JSON Schema, OpenAPI). Use for local inspection after SCL edits, and for required synchronization at integration/main after parallel work-item branches are merged.
---

# SCL 派生物の再生成

`ra` CLI が標準レイアウトから発見した `scl.yaml` が「単一上流」、HTML / JSON Schema /
OpenAPI はその「下流」。派生物は SCL からいつでも作り直せるため、並列 worktree では
commit タイミングを直列化地点へ寄せる。

## ブランチ別の扱い

- **work-item branch / 並列 worktree**: 確認用に再生成してよい。ただし生成物
  （HTML / JSON Schema / OpenAPI）は原則 commit しない。衝突した生成物は手で解かず、
  統合済み SCL から作り直す。
- **単独開発 branch**: 並列衝突の懸念がなければ、従来通り SCL と生成物を同じ commit に含めてよい。
- **integration branch / merge queue / main 直前**: 必ず再生成し、SCL と生成物を同期させる。
  main では `just scl-render` 後に差分が出ない状態を維持する。

## まず検証

```sh
just yaml-check-scl
```

## 一括再生成（推奨）

リポジトリルートから:

```sh
just scl-render
```

これは内部で標準 RA レイアウトを発見し、app と tool spec の派生物を再生成する。

## 仕上げ

再生成された成果物の diff を確認し、SCL の変更意図と一致しているかを見る。
integration / main へ入れる commit では生成物も含める（spec と生成物の同期を保つ）。
並列 work-item branch で確認用に生成した差分は、commit せず統合地点で再生成する。
