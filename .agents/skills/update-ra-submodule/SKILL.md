---
name: update-ra-submodule
description: Update a repository's Regenerative Architecture submodule and reconcile any required consumer-side changes. Use when a repo vendors this RA repository as a git submodule and the user asks to update, bump, sync, or refresh that submodule, especially when scl.yaml, spec/contexts, RA CLI behavior, generated artifacts, work-item conventions, or verification commands may need follow-up changes.
---

# RA submodule 更新フロー

RA を git submodule として使うリポジトリで、submodule の参照を進め、その更新に合わせて
利用側リポジトリの SCL・派生物・検証を同期する。

## 手順

1. **現状を確認する。**
   - `git status --short`
   - `git submodule status`
   - `.gitmodules` から RA submodule の path と branch 設定を確認する。
   - 未コミット変更がある場合は、ユーザーの変更として扱い、巻き戻さない。

2. **更新先を決める。**
   - ユーザーが commit / tag / branch を指定している場合はそれに従う。
   - 指定がなければ `.gitmodules` の branch を優先し、なければ submodule 側の現在 branch の
     upstream を使う。
   - ネットワークが必要な `git fetch` / `git submodule update --remote` が sandbox で失敗したら、
     承認付きで同じコマンドを再実行する。

3. **RA submodule を進める。**
   - 典型例:

     ```sh
     git submodule update --init <ra-path>
     git -C <ra-path> fetch --tags
     git -C <ra-path> checkout <target>
     ```

   - branch tracking 更新なら:

     ```sh
     git submodule update --remote <ra-path>
     ```

   - 更新前後の commit を控え、`git -C <ra-path> log --oneline <old>..<new>` と
     `git -C <ra-path> diff --stat <old>..<new>` で変更の性質を読む。

4. **利用側への影響を判定する。**
   - SCL 記法・必須フィールド・検証ルールが変わったら、利用側の `spec/scl.yaml` と
     `spec/contexts/*.yaml` を直す。仕様・振る舞いが変わる場合は `scl-change` Skill を使う。
   - `ra` CLI、`tools/*`、Just task、生成物形式が変わったら、利用側の `justfile`、CI、
     tool 呼び出し、生成済み HTML / JSON Schema / OpenAPI を確認する。
   - work-item / ADR / completion 形式だけの変更なら、既存ドキュメントに機械的破壊がないかを
     確認し、必要最小限の追随に留める。
   - RA の標準レイアウトは `spec/scl.yaml`、`spec/contexts/*.yaml`、`decisions/`、
     `work-items/`、`tools/*/spec/scl.yaml`。独自 registry ファイルは追加しない。

5. **SCL と派生物を検証する。**
   - まず `just yaml-check-scl` を実行する。
   - SCL を変更した、または RA の renderer / schema / OpenAPI 生成が変わった場合は
     `scl-render` Skill に従って `just scl-render` を実行する。
   - 生成物差分は SCL の意図と一致するか確認する。並列 work-item branch では生成物を
     commit しない運用があるため、ブランチの役割を確認する。

6. **アプリ側検証を走らせる。**
   - リポジトリの標準検証を優先する。例: `just verify-ui`、`go test ./...`、
     `bun test`、`bun run build`。
   - RA 更新による破壊なら、submodule の revision を戻して済ませず、利用側の仕様・呼び出し・
     生成物を新しい RA に合わせる。

7. **結果を報告する。**
   - RA submodule の old commit と new commit。
   - 利用側で変更したファイル。
   - 実行した検証と結果。
   - 残った未解決事項、またはユーザー判断が必要な互換性変更。

## 境界

- 明示依頼がない限り commit は作らない。commit を求められたら `commit` Skill を使う。
- submodule 内の RA 本体をこの流れで改変しない。必要なら RA 側の通常の work-item / SCL-first
  フローとして別作業に切り出す。
- 生成物の merge conflict は手で直すより、統合後の SCL から再生成することを優先する。
