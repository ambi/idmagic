---
status: completed
---

# main に対する Go race 検査を定期実行し、唯一の自動の race 検査にする

`mise run spec-diff` は main に対して規範の差分なしを示す。
`.github/workflows/idmagic-race.yaml` は、main の全体を `mise run test-go-race` で検査する唯一の自動の race 検査で、毎日 UTC 18:17 と手動の起動（`workflow_dispatch`）だけで動く。
push と Pull Request の CI は通常のビルドのままで、待ち時間を増やさず、race の失敗で PR を止めない。
数日に一回 push と PR の CI で race を有効にする案は、切り替えに履歴か日付の規則が要り、race の日の PR が無関係の既存の競合で止まるので採らなかった。
構成は Go 全体テストと同じく DB を必須にし、検証の証拠を失敗時も artifact に残す。失敗は GitHub が cron を最後に変更した利用者へ通知する。
`continuous-integration.md` に、失敗した実行と直前に成功した実行のコミットの範囲から原因を探す手順を書いた。
`check` のリポジトリのテストが、push と PR から起動しないこと、DB 必須、証拠の保存を確かめる。
手元で、データ競合を起こすテストを置くと race 付きの実行器が `DATA RACE` を検出して終了コード 1 を返し、race なしでは通ることを確かめた。
Actions 上での成功と、故意の race による失敗の確認は、push の後に利用者が行う。
