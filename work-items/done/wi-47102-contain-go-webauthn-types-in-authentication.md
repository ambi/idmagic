---
status: completed
---

# go-webauthn の型を authentication モジュールの内部へ閉じ込める

`mise run spec-diff` の結果は「no normative specification change against main」であり、規範の差分はない。
go-webauthn の `webauthn` パッケージの型を、本番コードでは WebAuthn のユースケースのパッケージだけに閉じ込めた。
公開ポートのセッションは、go-webauthn が直列化した JSON を不透明に持つ `WebAuthnSession` に替え、直列化と復元をユースケースへ移した。DB に保存する JSON の形は変わらず、旧いバージョンが書いた形で完了できることをテストで固定した。
RP は所有型 `RelyingParty`（組み立て地点からは `authentication.WebAuthnRelyingParty`）で包み、`cmd/internal/bootstrap` と `shared/http/server_http` から go-webauthn への依存を外した。使われていなかった `server_http.Deps.WebAuthnRP` は削除した。
ユースケースが返す options の `protocol` の型は、所有型にすると仕様の大きな構造の写しになるため、HTTP 層が import せずに JSON にする形のまま残した。
用語集にセレモニーを追加し、WebAuthn の機能文書に外部の型の閉じ込めと保存形の方針を書いた。
