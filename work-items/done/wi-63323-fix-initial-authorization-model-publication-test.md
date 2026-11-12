---
status: completed
---

# 認可モデル未作成時のテンプレート公開試験の失敗を解消する

規範仕様の差分はない（`spec-diff`：no normative specification change against main）。
失敗の原因は画面ではなく試験の待機の条件だった。
テンプレートを選んだ時点で編集欄に `document` が現れるため、`findByText(/document/)` は確認の表示を待たずに戻り、最新の読み直し（404）が先に終わったときだけ `Publish` が見つかっていた。
試験は確認の表示にだけある見出しの role を待つ形にし、確定前に登録の要求を送らないことの表明を加えた。
