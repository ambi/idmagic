---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 利用者ポータルのアプリ並び替えをドラッグアンドドロップで操作できるようにする

AccountApps の手動並び替え UI を、上下ボタンだけのリストから通常表示でグリップをドラッグできる
タイルグリッドへ更新した。カテゴリ別セクション表示と既存の `ReorderMyApplications` API /
ApplicationOrdering 永続化は維持し、dnd-kit の `DndContext` / `SortableContext` /
`DragOverlay` によってタイル左上グリップのドラッグでグローバル順序を操作する。ドラッグ中は
元タイルを薄く残し、ポインタに追従するプレビューを表示することで、何を動かしていて、
どこへ入るのかを分かるようにした。ドラッグ完了後の自動保存、保存中状態、可視の上下ボタン
削除、キーボード操作による代替並び替えを維持した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
