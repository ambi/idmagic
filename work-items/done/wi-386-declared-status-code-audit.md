---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# operation が宣言するステータスコード集合と handler が実際に返すものを総なめで突き合わせる

`mise run spec-diff` は 26 の TypeSpec 宣言の追加と 1 件の削除を返す。追加は 401 の共有応答
`AuthenticationRequiredResponse` と、404 / 409 / 410 / 503 の本体になる 25 のエラーモデルである。削除は
`FederatedLoginStartResponse` で、`StartFederatedLogin` が 303 リダイレクトしか返さないと確かめた結果、
それを指していた 200 の宣言ごと消えた。
`mise run check-status-drift` を足し、`mise run check` に組み込んだ。operation ごとに、生成 OpenAPI が宣言する
ステータス集合と、handler および guard が書くコードを突き合わせる。**検査が見つけた 293 件はすべて解決した。**
許可リストも閾値も置いていない。内訳は、宣言が足りない 275 件 (401×217、404×41、503×35、400×14、204×13、
500×10、303×8、201×8、403×7、409×3、410×1、304×1) と、宣言が余っている 18 件である。
`docs/api-rules.md` に「Declared status codes」節を足し、何を宣言し何を宣言しないかを決めた。共通の
error handler が写像しなかったエラーに対して書く 500 と、テナント解決 middleware が routing の手前で返す
404 `tenant_not_found` は operation ごとに宣言しない。前者はどの operation でも同じに出て呼び出し側の対応が
変わらず、後者はそもそもどの operation の応答でもないためである。401 は 403 と同じ guard の 2 つの分岐なので、
403 を宣言する operation は 401 も宣言する。
実ツリーは 333 operation 中 82 を全部読み、248 は読み残しあり、3 は経路または handler に届いていない。この
3 つの数は毎回の出力に書く。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
