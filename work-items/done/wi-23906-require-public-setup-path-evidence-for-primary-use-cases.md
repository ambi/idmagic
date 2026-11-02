---
status: completed
---

# 主要ユースケースの準備から利用結果までを証拠として要求する

利用者、クライアント、Application や接続を API で登録し、OAuth、三方式のサインオン、Agent、委譲、ワークロード認証、SCIM、下流 worker の利用まで確認するテストを追加した。
一時的な検証計画に準備方法を記す `setup` を加え、Go の完了検査では既存の実行報告と照合する。古い結果、Skip と未実行は受け付けない。
検出したい故障を注入して確認した。画面、実サービス、worker の別プロセス起動は今回のテストに含めない。
製品仕様の変更は前提 WI の SCIM 修正に限る。REQ-SOURCING-025 を追加し、REQ-SOURCING-004 と TypeSpec の `PatchScimUser`、`ScimResource` を更新した。
