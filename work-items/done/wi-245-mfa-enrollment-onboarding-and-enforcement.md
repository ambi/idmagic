---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# MFA 必須化時の登録オンボーディングと強制を分離する

- MFA 強制開始日、登録猶予、管理者承認バイパスをサインインポリシーへ追加し、未登録者数と
ロックアウト影響を管理画面から確認できるようにした。
- 未登録ユーザーは、期限内かつ管理者が発行した一回限りのバイパスがある場合だけ、パスワード
検証後に登録専用 pending session へ進む。TOTP 登録完了後は同じ LoginSession と OAuth2
transaction を昇格・継続し、それ以外は fail closed とした。
- バイパスの発行・取消・原子的消費・期限切れを memory/PostgreSQL adapter、管理 API/UI、監査
イベントまで実装し、認証途中の TOTP 登録画面と管理者復旧導線を追加した。
- 登録オンボーディングの trust anchor と初期 factor の判断を ADR-110 に記録し、SCL 派生物を
再生成した。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
