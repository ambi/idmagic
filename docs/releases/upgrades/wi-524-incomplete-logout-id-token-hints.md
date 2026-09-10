# WI-524: `id_token_hint` 付きログアウトの前提を確かめる

作業項目は `wi-524-reject-incomplete-logout-id-token-hints` である。

`/end_session` に `id_token_hint` を付けている RP は、そのヒントが `sub`、`aud`、`sid` を持つかどうかを確かめる。3 つのいずれかを欠くヒントは `invalid_request` の 400 で拒否されるようになった。

影響が出るのは、Device Authorization Grant または CIBA で受け取った ID Token をヒントにしている RP である。この 2 つの経路が発行する ID Token はブラウザーセッションに紐づかないため `sid` を持たない。これまではヒントとして受理され、ブラウザー Cookie が示すセッションが失効していた。この降格は無くなった。

対処は 2 つのいずれかである。認可コード交換で受け取った `sid` 付きの ID Token をヒントに使うか、`id_token_hint` を付けずに `/end_session` を呼ぶ。後者では Cookie による従来の解決がそのまま働く。`post_logout_redirect_uri` を使う場合は `client_id` を付ける。

`sub` が、`sid` の LoginSession の持ち主と違うヒントも拒否されるようになった。ログアウトのたびに ID Token を使い回している実装では、別の利用者がサインインしたあとに古い ID Token を送ると、この条件に当たる。ログアウト対象の利用者が現在のセッションの持ち主であることを、呼び出し側で確かめる。

拒否されたときに LoginSession と RefreshTokenRecord は失効しない。拒否を受けた RP は、ヒントを付け直して呼び直すか、ヒント無しで呼び直す。

データ移行と設定変更は要らない。`id_token_hint` の有効期限の扱いも変わっていないため、`exp` を過ぎたヒントはこれまでどおり受理される。

互換性境界は [REQ-OAUTH2-024](../../contexts/oauth2/scenarios.feature.md#rule-req-oauth2-024-rp-initiated-logout-は-id_token_hint-からセッションとクライアントを特定する) と [OIDC-LOGOUT-ID-TOKEN-HINT](../../contexts/oauth2/standards.md#openid-connect-rp-initiated-logout-10) が定める。
