# 一括転送

User、Group、Group のメンバーシップを、CSV のファイルで一括して取り込み、書き出す機能を扱う。
CSV は二つ目のプロビジョニングの権威ではなく、IdManagement が備える部分更新の窓口である。
外部の権威からの継続的な取り込みは `Sourcing` が、外部の宛先への送り出しは `Provisioning` が扱う。
本人によるアカウントデータのエクスポートは[アカウントのセルフサービス](../principals/account/README.md)が扱う。

| 文書 | 内容 |
| --- | --- |
| [CSV の転送](csv-transfer/README.md) | User と Group の CSV が共有する、見出し、行、セルの読み書きと転送の上限 |
| [ユーザー CSV](user-csv/README.md) | User の CSV のインポートとエクスポート |
| [グループ CSV](group-csv/README.md) | Group と、一つの Group のメンバーシップの CSV のインポートとエクスポート |
| [データエクスポート](data-export/README.md) | 非同期のエクスポートのジョブの開始、一覧、ダウンロード、取り消し、保持期限 |
