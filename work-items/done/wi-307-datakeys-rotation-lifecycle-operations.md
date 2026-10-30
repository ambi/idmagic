---
status: completed
---

# DEK の定期ローテーションと、制御面主体による緊急のローテーションと無効化の経路を追加する

`spec-diff` の差分は、REQ-DATAKEYS-007、008、009 の追加、REQ-DATAKEYS-002、003 の変更、`RotateTenantDataKey`、`DisableTenantDataKey`、`DataKeyVersionNotFoundError` の追加、`DataEncryptionKeyRotated`、`DataEncryptionKeyDisabled` への `actorUserId` の追加である。
DEK のローテーションと無効化に本番の経路ができた。
定期のローテーションは `idmagic-batch data-key-lifecycle`（毎日、周期 90 日）が行い、ジョブの投入に失敗した分も次の実行で回収する。
緊急のローテーションと無効化は制御面主体の対話セッションに限り、テナント管理者には公開しない。当初案から変えた判断で、`decisions.md` に残した。
破棄の経路、他プロセスの DEK の保持への伝播（wi-22156）、無効化した行が再暗号化を止める欠陥（wi-41449）は残る。伝播がない間の緊急の手順は再起動を含み、`backup-restore-dr.md` に書いた。
確認の限界として、PostgreSQL の構成での HTTP とバッチの通しの実行はしておらず、本番規模の所要時間も測っていない。リポジトリの実装は共有契約で確かめた範囲である。
