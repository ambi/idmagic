---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-307-datakeys-rotation-lifecycle-operations]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-003, impact: conforms }
---

# DEK の無効化とローテーションを、すべてのプロセスの DEK の保持へ伝える

## 動機

アンラップした DEK は、各プロセスのメモリ（`DataKeyCache`）に期限なしで保持する。
ライフサイクルの操作は、操作を実行したプロセスの保持だけを破棄する。
wi-307 がローテーションと無効化の経路を入れると、ほかの `api` と `worker` のプロセスは、再起動するまで退役したバージョンで暗号化し、無効化したバージョンで復号し続ける。
REQ-DATAKEYS-003 は、無効化したバージョンによる以後の復号の拒否を求めている。

wi-26063 で DataKeys の内部設計をコードと照合して見つけ、`docs/modules/data-keys/design/risks.md` に載せた。

あわせて、`docs/design/security/secrets.md` の DEK の生成の行は「テナントの初期化時に最初のバージョンを作る」と書くが、コードは項目を初めて暗号化するときに遅延して生成する（`FieldCipher.Encrypt`）。

## 対象範囲

- ライフサイクルの操作が、すべてのプロセスの DEK の保持に効く仕組みを入れる。
- `secrets.md` の DEK の生成の記述を、コードに合わせる。
- `docs/modules/data-keys/design/risks.md` の該当の行を消し、仕組みを `lifecycle/design.md` に書く。

## 対象外

- ローテーション、無効化、破棄の経路そのもの。wi-307 が扱う。

## 設計

候補は次のとおりである。着手時に一つに決める。

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| 保持に短い期限を設け、期限が来たらストアの状態を読み直す | 仕組みが単純で、プロセスの間の通信が要らない | 期限の間は無効化が効かない |
| 復号のたびにストアのバージョンの状態だけを読み、鍵素材だけを保持する | 無効化が即時に効く | 復号のたびにストアを読む |
| ライフサイクルの操作をプロセスの間へ通知する | 即時で読み取りも増えない | 通知の基盤が要る。製品は専用のイベントの基盤を持たない判断をしている |

## 計画

1. 案を決め、要件として残す結果（無効化がいつまでに全プロセスで効くか）を判断する。
2. 複数のプロセスを模したテストで RED を確かめ、実装する。

## タスク

- [ ] T001 [Spec] 伝播の方式と、残す結果を決める。
- [ ] T002 [App] 伝播を実装する。
- [ ] T003 [Docs] `secrets.md` と DataKeys の設計を直す。
- [ ] T004 [Verify] 検証する。

## 検証

- 二つの `DataKeyCache` を一つのストアに向け、片方で無効化した後に、もう片方の復号が拒否されることを確かめるテスト
- `mise run verify`

## リスク

保持に期限を設けると、マスターキーのプロバイダーへのアンラップの呼び出しが増える。
プロバイダーの流量の上限と照らして期限を決める。
