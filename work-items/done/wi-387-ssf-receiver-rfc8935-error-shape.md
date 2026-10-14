---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SharedSignals 受信エンドポイントのエラー本体を RFC 8935 §2.3 の形にする

`mise run spec-diff` は `no normative specification change against main` を返す。規範シナリオも標準行も動いておらず、変わったのは受信エンドポイントが返すエラー本体の欄名と、それを記述する 3 モデルである。`{"error": ..., "message": ...}` から RFC 8935 §2.3 が定める `{"err": ..., "description": ...}` へ変えた。この接点を Problem Details の外に置いている根拠が「標準が形を定めるから」であり、`docs/api-rules.md` も「各標準が定めるエラーレスポンスを返す」と書いていたので、文書が正しく実装だけが取り残されていた状態を解いたことになる。
**これは実際に線を越える変更である。** wi-397 が直した「返らない応答の宣言」とは違い、サーバーは本当に `{error, message}` を送っていた。`check-api-compat` は 6 件の破壊的変更を報告し (400 / 404 / 413 それぞれの `error` と `message` の削除)、この 6 件だけを凍結し直した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
