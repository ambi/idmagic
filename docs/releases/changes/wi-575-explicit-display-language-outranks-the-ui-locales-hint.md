# WI-575: 明示選択した表示言語を ui_locales ヒントより優先する

作業項目は `wi-575-explicit-display-language-outranks-the-ui-locales-hint` である。

WI-575 は、画面の表示言語を決める順序を「保存済みの明示選択 > `ui_locales` ヒント > ブラウザー言語 > 起動時のデフォルト」に変える。
これまでは `ui_locales` ヒントが保存済みの明示選択より先に評価されていた。

言語切り替え UI で `ja` を選んだ利用者に、RP が `ui_locales=en` を付けた認可リクエストを送っても、ログイン画面は `ja` で表示される。
言語を明示選択していない利用者には、従来どおり `ui_locales` ヒントが効く。

RP の開発者は、`ui_locales` が利用者自身の選択を上書きしないことに注意する。
OpenID Connect Core 1.0 も `ui_locales` を OP が従わなくてもよいヒントと定めている。

規範上の条件は [REQ-SYSTEM-008](../../domain/system/scenarios.feature.md#rule-req-system-008-oidc-の-ui_locales-ヒントにより表示言語が決まる) が定める。
