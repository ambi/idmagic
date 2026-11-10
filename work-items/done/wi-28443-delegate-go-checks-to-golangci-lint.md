---
status: completed
---

# tools の Go 検査を棚卸しし、golangci-lint に任せられる規則を移す

仕様の差分はない（`mise run spec-diff` は規範の変更なし）。
`domain` の作用の import は depguard の `domain-effects`、`domain` の `time.Now` と `bootstrap` 外の起動設定の読み取りは forbidigo へ移し、境界検査の `domain-effect` と環境変数の判定を削除した。
移行先は別名の import と関数値の参照も型情報で拒否する。
`testing` 区画が本番から到達するようになった場合の環境変数の読み取りは検出しなくなる。
層の外向き依存は depguard の接頭辞では表せないため境界検査に残し、import 宣言の外の文字列を依存に数えていた抽出を、モジュール間の依存と同じ import 抽出へ揃えた。
公開範囲、循環、共有ライブラリ、組み立て地点、テーブルの書き込み、SQL の直接呼び出し、その他の照合は、台帳、外部の宣言、受信側の型によらない判定を要するので残した。
