---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# サービス目標に安定 ID を与え、しきい値の複製と宙に浮いた参照を畳む

`docs/capacity.md` のサービス目標 16 件と容量受入れ目標 3 件に、`SLO-TOKEN-LATENCY` のような記憶しやすい安定 ID を与えた。連番ではなく語にした決め手はアラートの `summary` での読みやすさで、`SLO-LATENCY-004 を超過` より `SLO-TOKEN-LATENCY を超過` のほうが当番担当者に伝わる。Context 接頭辞は、非 5xx 比率と可用性の母集団が Context をまたぐため採らなかった。参照側は 4 か所すべてを ID 参照へ張り替え、`OAuth2/objective/*`・`oauth2.yaml`・`authentication.yaml` という実体の無い名前を消した。8 件のアラートすべてに `runbook_url` を付け、`page` 級が指す runbook 5 件を新しく書いた。`check-slo-references` が、資材の名指しする ID の実在と `page` 級の runbook の到達を強制する。**しきい値の数値は照合しない。** アラートは 5 分のバーンレート窓、目標は 30 日の移動窓で別のものを判定しており、一致の強制は「検査を通すために目標を動かす」誘因になる。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
