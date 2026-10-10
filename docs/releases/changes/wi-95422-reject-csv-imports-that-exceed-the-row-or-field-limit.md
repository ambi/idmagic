# WI-95422: 行数またはフィールドの上限を超える CSV のインポートを拒否する

作業項目は `wi-95422-reject-csv-imports-that-exceed-the-row-or-field-limit` である。

WI-95422 は、User、Group、メンバーシップの CSV のインポートで、実効の転送ポリシーの `max_rows` または `max_field_bytes` を超えるファイルを、投入の時点で拒否する。

これまでは `max_bytes` だけを投入の時点で拒否し、行数と項目長の超過はプレビューのジョブの中で検出していた。
ジョブは上限に達する前の行を計画したうえで成功し、その適用は先頭の行を取り込んだ。
変更後は、プレビューの投入が 400 と `too_many_rows` または `field_too_large` で拒否され、プレビューのジョブは作られない。

プレビューの後に実効の上限が下がり、保存したファイルが上限を超えた場合、適用のジョブは一行も確定せずに失敗する。
規範上の条件は[ユーザー CSV](../../modules/identity-management/user-csv/README.md)の REQ-IDMANAGEMENT-004、[グループ CSV](../../modules/identity-management/group-csv/README.md)の REQ-IDMANAGEMENT-026 と REQ-IDMANAGEMENT-029 が定める。
