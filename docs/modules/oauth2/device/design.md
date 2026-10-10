# デバイス認可の設計

この文書は、[デバイス認可](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

### デバイス認可グラント

デバイスフロー（RFC 8628）、すなわち `POST /device_authorization`、`/device` の検証 UI、`/token` の `device_code` グラントでは、承認・拒否・交換の遷移をその場で再実装せず、`DeviceCodeFlow` の遷移表を共有する 1 つの遷移関数を使う。`device_code` は 32 バイトのランダム値であり、ベアラーシークレットとして SHA-256 ハッシュだけを保存する。`user_code` は母音と見分けにくい文字を除いた 20 文字の縮小済みで曖昧さのない文字集合を使い、`WDJB-MJHT` のようにグループ分けして表示する。ポーリングは仕様中核で定める間隔とバックオフ増分に従い、`authorization_pending` / `slow_down` / `access_denied` / `expired_token` を返す。二重発行を防ぐため、承認済みコードをトークン発行前に `approved → exchanged` へ遷移させる。
