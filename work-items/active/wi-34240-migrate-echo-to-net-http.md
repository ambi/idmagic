---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-07
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "HTTP の経路、メソッド、状態コード、ヘッダー、本文、Cookie、リダイレクト先、CSRF とレート制限の判定を変えない。ルーティングの挙動差を仕様として受け入れる必要が生じた場合は仕様の段階へ戻る。" }
---

# HTTP の配線を Echo から標準 net/http へ移す

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、Echo を標準 `net/http` へ移すと決めた。
Echo v5.4.2 が追加で持ち込むのは golang.org/x/net と x/time（Go チーム）だけで、依存の数では差がない。
差は、信頼する公開元（labstack）が一つ減ることと、Echo のルーティングとバインドの入力処理が経路から外れることである。
Gin は依存が増えるため比較対象から外した。

## 対象範囲

- 本番コードの 173 ファイルが使う Echo の機能を、標準の `http.ServeMux` と小さな補助へ移す。
  使っている機能は `Group`（42 ファイル）、`Param`（49）、`Get`/`Set`（67）、`NoContent`（44）、`QueryParam`（25）、`JSON`（18）、`FormValue`（11）、`Redirect`（10）、`Cookie`（9）、`Stream`/`Blob`（8）、中央のエラー処理、`Bind`（3）、ミドルウェアの `BodyLimit`（1）である。
- エラーを返すハンドラーを `http.Handler` へ変換するアダプター、JSON 応答、グループ化の補助を、`backend/shared/http/` に置く。
- クライアント IP の抽出を一つにする。
  現在は `support_http.ExtractClientIP` と、`oauth2` と `signin` の非公開の複製の計 3 か所にある。
  Echo の `RealIP()` は使っておらず、信頼する段数だけ X-Forwarded-For をさかのぼる自前の処理である。

## 対象外

- 経路、状態コード、本文の形の変更。
- Gin など別のフレームワークへの移行。

## 設計

標準の ServeMux（Go 1.22 以降）は、メソッドとワイルドカードを含むパターンで経路を選び、`Request.PathValue` で値を取り出す。
416 ルート相当の合成ルートで計測すると、1 リクエストのルーティングは ServeMux で約 0.1〜0.2µs、Echo で約 0.03〜0.07µs だった（Apple M3）。
差は最も厳しい SLO（`/introspect` の p99 ≤ 50ms）に対して無視できる。

Echo と挙動が異なり得る点を、移行前に契約テストで固定する。

| 挙動 | ServeMux のデフォルトの振る舞い |
| --- | --- |
| 正規化していないパス（`//`、`..`、末尾スラッシュの違い） | 正規化したパスへ 301 でリダイレクトする |
| メソッドの不一致 | 405 と `Allow` ヘッダーを返す |
| HEAD | GET のパターンが HEAD にも一致する |
| パーセントエンコードされたセグメント | セグメントごとに復号して照合する |
| パターンの重複 | 登録時に panic する |

現在の応答と異なる挙動は、補助のレイヤーで現在の応答に合わせるか、仕様の段階へ戻って受け入れるかを個別に決める。
`Get`/`Set` によるリクエスト単位の値は `context.Context` に移し、キーの型を非公開にする。
`Bind` の 3 か所は、型付きの JSON デコードと、既存の検証の呼び出しに置き換える。
`BodyLimit` は `http.MaxBytesReader` に置き換え、413 の応答を保つ。

## 計画

1. クライアント IP の抽出を一つにする構造変更を先に行う。
2. ルーティングの挙動差の表を契約テストに固定する。
3. 補助を作り、モジュールごとに移す。各段階で契約テストと E2E を通す。
4. 最後に Echo を依存から外す。

## タスク

- [ ] T001 [Refactor] クライアント IP の抽出を一つにする。
- [ ] T002 [Characterize] ルーティングの挙動差を契約テストに固定する。
- [ ] T003 [Adapters] エラー処理、JSON 応答、グループ化の補助を作る。
- [ ] T004 [Adapters] モジュールごとにハンドラーとミドルウェアを移す。
- [ ] T005 [Verify] Echo を依存から外し、依存グラフと性能を確かめる。

## 検証

- 移す各モジュールで `mise run test-go-package -- <package>` を実行する。
- HTTP の契約は `mise run check-route-reference`、`mise run check-gateway-routes` と HTTP 統合テストで確かめる。
- 全体は `mise run verify-full` で確かめる。

## リスク

経路の照合や正規化の差で、CSRF、Origin の検証、レート制限の前提が崩れる可能性がある。
挙動差を先にテストで固定し、モジュールごとに移す。
