# SharedSignals Internals

## CheckRevocationEpoch
指定した Agent の失効エポックを返す。OAuth2 の Introspect または保護 API がアクセストークンの `issued_at` と比較し、失効エポック以前に発行されたトークンをフェイルクローズで無効と判定するために呼び出す。

## ResolveSecurityEventSubject
受理した SET の `events` クレームから、失効を反映する対象のテナントローカルなプリンシパルを決める。IdMagic 自身の送信側が生成する独自形式 (`subject_type` / `tenant_id` / `principal_id`) と、RFC 9493 の Subject Identifier (`format=iss_sub` と `format=opaque`) の両方を解釈する。判別は `format` メンバーの有無で行い、両形式を同時に満たす表現は受け付けない。RFC 9493 形式は自身のテナントを名乗らないため、テナントは受信ストリームが属するテナントで決まる。識別子は Agent の識別子として解決し、一致しなければ Agent に束縛済みの `OAuth2Client` の識別子として解決する。どちらでも解決できない場合はフェイルクローズで拒否する。
- Result invariant: subject_resolves_within_receiving_stream_tenant(input.stream_id)

## AdvanceRevocationEpoch
KillAgent、DisableAgent、UnbindAgentCredential、所有者（`owner_user_id`）の無効化または削除、受理済みの SecurityEvent のいずれかを契機として、対象 Agent 群の失効エポックを進める。所有者のオフボーディングでは、対象テナント内で `owner_user_id` が一致するすべての Agent を一括して進める。失効エポックを既存値より前の時刻に戻すことはない。
- Result invariant: epoch_advances_monotonically_for_all(input.agent_ids)
