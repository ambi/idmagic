---
status: completed
archived_from: 49c4639343f51d17c069cef2762ae9d546e822ee
---

# 残りの Context の仕様と設計を、機能仕様と内部設計の構造へ移す

`mise run spec-diff` は、main に対する規範仕様の差分を報告しない。
残りの 19 の Context を、IdManagement と Tenancy と同じ機能仕様と内部設計の構造へ移し、旧形式のまま残る Context はなくなった（`tools/check/legacy-spec-layout.json` の一覧は空である）。
各 Context の README は責務と境界、モデル、公開する契約、機能の索引からなり、規則は機能ノードの機能仕様へ、例は付録へ、判断と仕組みは機能の `design.md` と `design/`（アーキテクチャ、重要な判断、話題ごとの文書、横断的概念）へ振り分けた。
Authentication、OAuth2、Sourcing はコードの機能スライスに合わせて機能ノードを置き、`tools/check/feature-node-debt.json` の未対応のスライスは空になった。Authentication のルートの規則はログインとアカウントポータルの機能ノードへ移した。
要件の題名、本文、例、状態表は変えていない。`Primary actor:` だけの要件は要件文を持たないまま移し、書き直しは wi-35451 に委ねた。
旧内部設計のうちコードと食い違う記述はコードに合わせて書き、食い違いを各 Context の `design/risks.md` に載せた。主なものは、DataKeys の DEK のローテーションと無効化と破棄を呼ぶ経路がないこと、Seeding、IdGovernance、Sourcing が記録の正を持つ Context の Repository を直接書くこと、Audit の 7 年の保持の目標と保持期間の掃引の食い違い、SharedSignals の失効エポックの反応が `worker` のイベントに働かないこと、SigningKeys の XML の署名の鍵が自動で回らないことである。
旧形式の経路の除去は wi-53794 へ、用語とファイルの名前の見直しは wi-99596 へ分けた。

全文は `git show 49c4639343f51d17c069cef2762ae9d546e822ee:<このファイルのパス>` で読む。
