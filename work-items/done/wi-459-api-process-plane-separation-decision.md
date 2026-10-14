---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# API プロセスを当面分けない判断と再検討条件を、容量シナリオと隔離要件に基づいて記録する

`mise run spec-diff` は `no normative specification change against main` を返す。規範的シナリオ、標準要件、TypeSpec 宣言のいずれも追加、変更、削除していない。意味上の差分は正準文書の側にあり、次の 3 つである。第一に `docs/capacity.md` に `Non-protocol request profile` を追加し、管理コンソール、管理 API 自動化、ポータル、SCIM、Shared Signals の受信について、18 個の入力とそこから導いた通常時、最繁時、集中実行時の到達率を Planning assumption として置いた。最繁 15 分の合計は中央値 1,550 rps、上界 2,710 rps で、認証・プロトコル系 41,250 rps に対して 4–7% にあたる。第二に同書の Sizing rules へ、種別ごとの 1 レプリカ当たり持続処理能力と、種別ごとに実行単位を分ける構成を同じ式で評価する方法を加えた。分離に固有の常時増分は総需要ではなく可用性下限の重複、`ceil` の重複、余裕を融通できないことの 3 つから生じ、最小の分離でも +3 レプリカ、+48 論理接続、利用可能接続の必要量 +69 が下限になる。第三に `docs/domain/system/decisions.md` へ `No API plane separation` を追加し、単一の API Deployment を維持する判断、却下した 3 つの案、再検討する条件を記録した。`docs/deployment.md` と `docs/domain/structure.md` からはこの節を参照するだけにして内容を複製していない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
