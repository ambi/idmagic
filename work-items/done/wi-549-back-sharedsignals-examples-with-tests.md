---
status: completed
archived_from: 52dc31634dd359e4e95e6662696b21f7c69e2bb8
---

# SharedSignals が宣言する具体例 20 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して `no normative specification change` を返した。TypeSpec の変更はイベント 1 件の説明文だけであり、契約の形は変えていない。
SharedSignals が台帳に残していた 20 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
既存テストへディレクティブを足すだけで済んだのは、RFC 9493 の Subject Identifier を扱う 6 件（`EX-SHAREDSIGNALS-010-01` から `-06`）だけだった。この既存テストは 6 件の入力、失効エポック、`SecurityEventReceived`、`rejected_subject_unresolved` をすでに観測している。同じ箇所で、過去の一括移行が途中で切っていた `RFC9493-SUBID-ISS-SUB` のディレクティブの文も元の意味へ復した。
配送の 2 件（`EX-SHAREDSIGNALS-006-01`、`-02`）は `backend/sharedsignals/usecases/deliver_test.go` で消化した。006-01 は既存テストへイベント列全体、失敗ごとの試行回数、再試行までの間隔が伸びること、dead_letter に次の試行時刻が残らないことを足し、006-02 は 3 回目で配送が成功する事例を新しく書いた。
残る 12 件は `backend/sharedsignals/handlers_http/scenario_examples_test.go` を新設し、製品と同じ `server_http.Register` の組み立てで観測した。Agent の強制終了は IdManagement の管理 API、トークンは `/token` と `/introspect`、SET は外部の送信者の鍵で実際に署名して受信エンドポイントへ送る。拒否の効果は失効エポック、ストリーム、付随する設定の保存回数、配送、クォータの利用量、発行イベントから読み直している。
照合で分かった食い違いは 1 件だった。配送の最終失敗で `SecurityEventDeliveryFailed` を発行していなかった。`states.md` の遷移表（`pending →(Failed)→ failed →(DeadLettered)→ dead_letter`）とも食い違っており、実装の欠陥と判定して利用者の指示で本項目で直した。TypeSpec の同イベントの説明だけが「a retry was scheduled」と限定していたので、具体例と `states.md` に合わせて改めた。

全文は `git show 52dc31634dd359e4e95e6662696b21f7c69e2bb8:<このファイルのパス>` で読む。
