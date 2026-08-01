---
status: accepted
authors: [tn]
created_at: 2026-08-01
supersedes: [ADR-125]  # 自動 sunset を標準管理方式とする判断のみ
---

# ADR-152: client secret を追加発行と個別失効で管理する

## コンテキスト

ADR-125 は client secret の無停止切替を可能にしたが、rotation 実行時に既存 credential の終了時刻も
同時に決める。そのため管理者は現在の credential 一覧と状態を確認できず、連携先の切替完了を確認して
から旧 credential だけを失効する運用もできない。実際の管理画面では、通常設定とは性質の異なる即時
credential 操作が同じ保存 form 内に置かれ、境界も不明瞭になった。

一方、任意個数の credential を許すと履歴・認証コスト・削除判断が無制限に増える。無停止切替に必要な
同時有効数は2件のままで足りるため、件数制約と操作モデルは分けて決める必要がある。

## 決定

- 通常の管理方式を、既存 credential を変更しない期限付きの追加発行と、credential ID を指定する個別
  失効へ分ける。従来の rotation interface は互換性のため残すが、新しい管理 UI では使用しない。
- Active credential は最大2件とし、並行発行でも超えないよう保存境界で原子的に保証する。
- 新規発行 credential は期限を必須とする。既存の期限なし credential は稼働中 client を停止させない
  ため grandfather し、最初の追加発行時に必要なら credential table へ backfill する。
- Application を唯一の編集面とし、secret の平文は発行成功応答で一度だけ返す。管理 UI は通常設定の
  保存 form 外に独立した credential 管理セクションとして置く。
- 発行・失効は別々の非機密監査イベントとして記録する。

対応する正本は `spec/contexts/oauth2.yaml` の `models.ClientSecretCredential`、
`states.ClientSecretCredentialLifecycle`、client secret lifecycle scenario と、
`spec/contexts/application.yaml` の `interfaces.IssueApplicationClientSecret`、
`interfaces.RevokeApplicationClientSecret` である。

## 却下した代替案

- **ADR-125 の自動 sunset だけを維持する**: 切替完了を管理者が確認してから旧 credential を失効する
  運用を表現できず、一覧・状態表示を追加しても操作と一致しない。
- **Active credential を任意個数にする**: 通常の無停止切替に不要で、漏洩面・認証コスト・運用判断を
  無制限に増やす。
- **既存の期限なし credential に一律期限を設定する**: 移行時点から将来の認証停止時刻が生まれ、
  管理者が連携先を更新していない client を壊し得る。
- **通常設定の保存 form 内に credential 操作を残す**: 保存ボタンで反映される設定と即時実行される
  破壊的操作の境界が一致せず、wi-314 で起きた配置上の誤解を再発させる。

## 影響

- ADR-125 は accepted のまま、current credential と旧 credential の自動 sunset を標準 UI とする
  判断だけが本 ADR に置き換わる。最大2件、Application が唯一の編集面、秘密値を一度だけ返す判断は
  維持する。
- OAuth2/Application SCL に状態、追加発行・個別失効 contract、監査イベント、管理画面 flow が増える。
- 保存 adapter は上限検査と発行を同一 transaction にし、UI は独立したトップレベルセクションを持つ。
