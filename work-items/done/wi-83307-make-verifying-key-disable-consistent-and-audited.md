---
status: completed
---

# 検証用の署名鍵の無効化を、構成によらず同じ結果にし、監査へ残す

REQ-SIGNINGKEYS-010 を更新し、無効化した非 Active の鍵を Archived にして鍵素材を保持し、公開から除く動作をメモリ、PostgreSQL、Vault で揃えた。
状態遷移と TypeSpec に SigningKeyDisabled を定義し、保存後に occurredAt、tenantId、kid を発行して監査へ残す。
排他下で変更有無を返すため、再要求と並行要求でイベントを重複させず、拒否と保存失敗では発行しない。
共有契約、管理 API からの監査の読み戻し、保存先の race 検査、verify-changed を行った。
ローテーションと退役の変更は対象外とし、ブラウザー E2E と専用の外部環境試験は実行していない。
監査は既存の保存後の同期配信を使うため、保存と配信の間の停止による欠落は残る。
