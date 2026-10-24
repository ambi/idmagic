---
status: completed
---

# サインインポリシーの定義と評価を Application から分ける

サインイン条件の定義、検証、デフォルトと上書きの選択、評価、保存と更新イベントを SignInPolicy へ移した。
Application はアプリの実在、有効性と割り当ての関門を担当し、Authentication と各プロトコルは SignInPolicy の公開契約を使う。
アプリの存在確認は要求側の `ApplicationLookup` を組み立て地点で結び、SignInPolicy から Application を import する依存を作らない。
ポリシー表の所有者と sqlc の生成先を移し、保存する JSON、SQL のテナント確認、外部キーの cascade、アプリ削除の順序を維持した。
外向きの Application の API と要件 ID、MFA、信頼済みデバイス、ネットワークと再認証の意味は変えず、規範仕様の差分はない。
再認証の時刻の境界を既存の判定に合わせて固定し、移設前から呼び出し側にあるテストでも変異を調べられるよう、検証タスクに `integration` モードを加えた。
