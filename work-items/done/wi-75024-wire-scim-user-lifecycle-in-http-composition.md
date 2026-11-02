---
status: cancelled
---

# 正式な HTTP の組み立てで SCIM の User ライフサイクルを配線する

HTTP テスト構成の SCIM の無効化で nil の呼び出しを再現したが、本番の起動処理 `backend/cmd/idmagic/server.go` の `sourcingModule` は既に `UserLifecycleCommands` と失効の反応器を渡していた。
製品の未配線という前提が誤っていたため、本項目を中止する。
試した製品ルーターの変更は採用しない。
元の wi-23906 でテスト構成を本番の構成に合わせ、公開のライフサイクルの結果を検証する。
