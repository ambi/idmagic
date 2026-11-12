---
status: completed
---

# 古いバージョンを基にした認可モデルの発行を拒否する

`spec-diff` の差分は、REQ-AUTHORIZATION-018 の追加、REQ-AUTHORIZATION-011 の変更、`AuthorizationModelVersionConflictError` の追加、`PutAuthorizationModel` の変更である。
認可モデルの登録は必須の `expected_version`（未登録なら 0）を受け、最新と一致しなければ 409 と `authorization_model_version_conflict` で拒否し、バージョンもイベントも残さない。
欠落と負の値は 400 と `invalid_request` にした。初回公開前なので、省略を受理する互換の経路は設けていない。
比較と採番は一つの INSERT の条件で行い、同時に条件を通った挿入は `(tenant_id, version)` の一意制約の違反を競合へ変換する。
8 並列の共有契約ではこの違反の経路をほとんど通らないので、先の挿入をコミットせずに保つ PostgreSQL 専用のテストで、変換を外す故障を検出できることを確かめた。
管理コンソールは差分を比べたバージョンを送り、競合では確認を閉じて入力を残し、確認し直すよう示す。
競合の拒否は監査にもイベントにも残さない。状態を変えない拒否であり、422 の検証の拒否と同じ扱いである。
