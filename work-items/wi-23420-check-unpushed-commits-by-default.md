---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-08
priority: p1
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: >-
    リポジトリ検査の比較基準の既定値だけを変える。プロダクトの API の応答、永続状態、
    ドメインイベント、外部への呼び出しはどれも変わらず、検査が適用する規則も同じままである。
---

# リポジトリ検査の比較基準を、既定でまだ push していないコミットの起点にする

## 動機

CI の `Verify specification, backend, and UI` ジョブが、直近の push のたびに別のコミットで `check-repository` に失敗している。
失敗の内容はどれも同じで、`Spec-Impact` トレーラーの理由が、変わらない結果、状態、イベント、呼び出しを名指していない、というものである。
例: `Spec-Impact: none` だけで理由のないコミット（62be80f4、2f931a70）や、理由が規則を満たさないコミット（7710e0ed、4eb7dc41、4771c56f、ff7f8040）。

CI は `github.event.before` から `HEAD` までを検査する。
一方、`mise run check-repository` と `mise run check-spec-impact` の比較基準は既定で `main` である。
`main` 上で直接コミットすると範囲 `main..HEAD` が空になり、手元の `mise run verify` はトレーラーを一つも検査しないまま合格する。
違反は push の後、CI で初めて見つかる。
しかも CI は push ごとの範囲しか検査しないので、次の push では同じ違反がもう報告されず、失敗の記録だけが残る。

## 対象範囲

- 引数がないとき、上流ブランチ（`@{upstream}`）があればそれを比較基準にし、なければ従来どおり `main` にする。
- 同じ既定値を、比較基準を取る他のリポジトリ検査のタスクにも揃える。

## 対象外

- Git フック（`commit-msg`、`pre-push`）の導入。フックはワークツリーごとの設定が必要で、入れ忘れると黙って効かない。既定値の修正なら `mise run verify` を走らせるだけで効く。
- 過去のコミットのトレーラーの書き換え。すでに push 済みであり、履歴は書き換えない。
- `Spec-Impact` の理由に求める規則自体の変更。

## 設計

`main` 上で作業し、`origin/main` へ push する運用では、まだ push していないコミットの起点は `origin/main` である。
`@{upstream}` はこれを指し、作業ブランチでも、そのブランチの上流を指す。
上流のないブランチや新規のクローンでは `main` へ戻す。
これで、手元の検査範囲が CI が次の push で検査する範囲と一致する。

比較基準の解決は複数のタスクが使うので、`mise.toml` の各タスクへ個別に書かず、一か所にまとめる。

## 計画

1. 比較基準を取るタスクを `mise.toml` から洗い出す。
2. 理由のない `Spec-Impact: none` のコミットを手元で作り、`main` 上で `mise run check-repository` が合格してしまうことを確かめる（RED）。
3. 既定値を変え、同じ状態で失敗することを確かめる。

## タスク

- [ ] T001 [Acceptance] `main` 上の未 push のコミットを手元の検査が見逃すことを確かめる。
- [ ] T002 [Tooling] 比較基準の既定値を上流ブランチへ変え、比較基準を取るタスクで揃える。
- [ ] T003 [Docs] 比較基準の既定値を説明する開発文書を更新する。
- [ ] T004 [Verify] `mise run verify` を通す。

## 検証

- 理由のない `Spec-Impact: none` を持つ未 push のコミットがあるとき、`main` 上の `mise run check-repository` が失敗する。
- 上流のないブランチで、従来どおり `main` を比較基準にする。
- `mise run verify`

## リスク

リスクは low。
上流ブランチが `main` から大きく離れた作業ブランチでは、検査範囲が従来より狭くなる。
CI のプルリクエスト検査は `pull_request.base.sha` を明示的に渡すので、この変更の影響を受けない。
