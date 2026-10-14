---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# フロントエンドのアーキテクチャを独立した設計文書にし、UI 設計を書き足す

フロントエンドの設計が、`docs/design/application/frontend.md` と `docs/design/application/user-interface.md` の二文書に分かれて置かれるようになった。
`frontend.md` は、コードの置き場所を決める判定表、ディレクトリ間の依存の向き、機能スライスどうしを参照しない規則、`admin-users` の木構造の実例、機能スライスと Bounded Context が一致しない理由、ルーティングと認証ガード、サーバー状態のキャッシュを持たない選択と状態の置き場所、コンテナと表示用コンポーネントの分割、ビルドと配信、テストの置き場所を持つ。
`user-interface.md` は、情報構造とナビゲーションに加えて、表示状態、破壊的な操作、フォームと検証、一覧、エラーの文言、国際化、`docs/domain/standards.md` の WCAG 2.2 の各行を満たす実装手段を持つ。
`frontend/README.md` と `frontend/src/features/README.md` にあった設計と開発時の作業の記述は `frontend.md` と `user-interface.md` へ移し、README には文書へのリンクとよく使うタスクだけを残した。
レビューを受けて、二文書の冒頭で対象が Web フロントエンド・アプリケーションであることと対象の画面を示し、規則を表と番号付きリストに組み直し、採らなかった案を独立した表にまとめた。
`DOCUMENTATION_GUIDE.md` には、コードのディレクトリの README は入口に留め、`docs/` の文書は内容を README に委ねないという規則を加えた。
README は削除ボタンに `variant="outline" tone="danger"` を使うと書いていたが、ボタンに `tone` はなく、実装は `variant="destructive"` を使っていたので、実装に合わせて書いた。
`docs/domain/structure.md` のフロントエンドの節はディレクトリの表と依存の向きに絞り、論理アーキテクチャ、ランタイムアーキテクチャ、アプリケーション設計の索引、`DOCUMENTATION_GUIDE.md` §4.12 からの参照を加えた。
正準文書の閉じた集合に `frontend.md` を加えた。
Motivation はサーバー状態を `usePaginatedList` のようなフックが持つと書いていたが、観測するとこのフックと `LoadMoreButton` はどの画面からも使われておらず、一覧のページングは検索パラメーターと `PageNavigation` が持っていた。
規則に従わない現行の箇所は、文書には書かず、意図の確認を含めて work item に起票した。
機能スライス間の参照と `lib/` から `components/` への参照は [[wi-607-frontend-import-direction]]、使われていないページング補助は [[wi-608-remove-unused-paging-helpers]]、`window.confirm` による確認 5 か所は [[wi-609-replace-window-confirm-with-dialogs]]、`useFormatters` を通さない日時の整形は [[wi-610-format-dates-through-locale-formatters]] である。
依存の向きを検査する仕組みはなく、その旨を文書に書いた。
`standards.md` の見出しへのフラグメント付きリンクは、描画された仕様ページでアンカーが解決されず `check-rendered-spec` が拒否したので、フラグメントなしのリンクにした。
`mise run spec-diff` は規範の変更を報告しない。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
