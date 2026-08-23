# SharedSignals Internals

## The revocation epoch as the point of comparison

失効は個々のトークンを追いかけて消すのではなく、Agent ごとの 1 つの時刻として持つ。OAuth2 の Introspect と保護 API はアクセストークンの `issued_at` をこの時刻と比べ、それ以前に発行されたトークンをフェイルクローズで無効と判定する。発行済みトークンを列挙せずに全部を一度に無効化できるのは、この 1 つの時刻に集約しているからである。

## Resolving the subject of an inbound SET

受理した SET の `events` クレームから、失効を反映する対象のテナントローカルなプリンシパルを決める。IdMagic 自身の送信側が生成する独自形式 (`subject_type` / `tenant_id` / `principal_id`) と、RFC 9493 の Subject Identifier (`format=iss_sub` と `format=opaque`) の両方を解釈する。判別は `format` メンバーの有無で行い、両形式を同時に満たす表現は受け付けない。RFC 9493 形式は自身のテナントを名乗らないため、テナントは受信ストリームが属するテナントで決まる。識別子は Agent の識別子として解決し、一致しなければ Agent に束縛済みの `OAuth2Client` の識別子として解決する。どちらでも解決できない場合はフェイルクローズで拒否する。解決の探索範囲は受信ストリームが属するテナントを出ないので、他テナントのプリンシパルを名乗る SET が到達しても、そのテナントには作用しない。

## What advances the epoch

KillAgent、DisableAgent、UnbindAgentCredential、所有者（`owner_user_id`）の無効化または削除、受理済みの SecurityEvent のいずれかを契機として、対象 Agent 群の失効エポックを進める。所有者のオフボーディングでは、対象テナント内で `owner_user_id` が一致するすべての Agent を一括して進める。失効エポックは単調にしか進まない。既存値より前の時刻へ戻す経路が無いことが、遅れて到着した SET や再送によって一度失効させたトークンが再び有効にならないことを保証する。
