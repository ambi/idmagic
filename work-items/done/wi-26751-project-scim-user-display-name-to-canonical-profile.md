---
status: completed
---

# SCIM User の displayName を正規プロフィールの名前へ投影する

SCIM の作成と置換で失われていた `displayName` を正規の名前へ投影し、既存の文字列の `name.formatted` を空文字列も含めて優先する。
REQ-SOURCING-025 に優先順位、null、型の拒否、読み戻しと Discovery の対応を定め、REQ-SOURCING-004 と TypeSpec の `PatchScimUser`、`ScimResource` を同期した。
PATCH の二つの名前は同じ更新先を使い、公開資格情報による作成から管理 API までと、型の不正な操作列の無変更を検証する。
別の名前の保存先は増やさず、既存の検索は `name.formatted` を使う。
