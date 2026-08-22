package domain

// PrincipalTypeAgent は access token の `principal_type` claim が Agent を表す値。
// 発行側 (tokens_jose) と判定側がこの 1 つの定数を共有する。
const PrincipalTypeAgent = "agent"

// DelegationMode はトークンが今どの立場で振る舞っているかの区別。
// 新しい永続状態ではなく `act` チェーンと principal 種別から導出する派生値なので、
// クレームと食い違う第二の真実は生まれない。
type DelegationMode string

const (
	// DelegationModeDirect は代行が無く、subject が人間の利用者である場合。
	DelegationModeDirect DelegationMode = "direct"
	// DelegationModeAutonomous は代行が無く、subject 自身が非人間のプリンシパル
	// (Agent または client) である場合。エージェントの自律実行がこれに当たる。
	DelegationModeAutonomous DelegationMode = "autonomous"
	// DelegationModeOnBehalfOf は `act` に subject と異なる行為者がいる場合。
	// エージェントが利用者を代理している状態を表す。
	DelegationModeOnBehalfOf DelegationMode = "on_behalf_of"
)

// DelegationSubject は委譲モードの導出に必要な入力。トークンのクレームから
// そのまま埋められる形にしてあるので、introspection と監査が同じ材料を渡せる。
type DelegationSubject struct {
	// Sub はトークンの subject。
	Sub string
	// ClientID はトークンを提示しているクライアント。sub と一致すれば
	// 利用者の居ない machine-to-machine のトークンである。
	ClientID string
	// PrincipalType は `principal_type` claim。Agent に束縛されたトークンでのみ
	// PrincipalTypeAgent が入る。
	PrincipalType string
	// Act は RFC 8693 §4.1 の actor claim。
	Act map[string]any
}

// DeriveDelegationMode は委譲モードを導出する。導出をこの 1 関数に閉じることで、
// introspection の応答と監査イベントが同じ規則を通ることを保証する。
// 2 箇所に書くと両者が食い違い、しかもその不整合は調査のときに最も見つけにくい形で現れる。
func DeriveDelegationMode(in DelegationSubject) DelegationMode {
	if actChainDelegates(in.Act, in.Sub) {
		return DelegationModeOnBehalfOf
	}
	if in.PrincipalType == PrincipalTypeAgent || in.Sub == "" || in.Sub == in.ClientID {
		return DelegationModeAutonomous
	}
	return DelegationModeDirect
}

// maxActChainScan は act チェーンを走査する段数の上限。JSON の復号は循環を作れないが、
// この関数は Go で組み立てた map も受け取るので、判定が止まらない入力を構造的に排除する。
// 上限に達したチェーンは、深さの検査が別途拒否する側の入力なので代行として扱う。
const maxActChainScan = 64

// ActorChainParticipants は act チェーンに現れる主体を、外側 (現在の行為者) から
// 内側へ並べ、最後に subject を置いて返す。監査はこの並びを多値の検索属性
// `delegation.actor` として保存し、チェーンのどの段の参加者からでも同じ連なりを引く。
//
// 深さの整数だけでは「どこまで委譲されたか」は分かっても「誰を経由したか」は分からず、
// 中間の行為者を後から辿れない。チェーンに ID を振らないのは、参加者が act から導出
// できる以上、採番した ID は委譲モードの導出と食い違いうる第二の真実になるためである。
//
// 同じ主体が複数の段に現れても 1 度しか並べない。workload identity の自己交換では
// sub と act.sub が同じクライアントになるので、重複を残すと参加者が水増しされる。
func ActorChainParticipants(act map[string]any, sub string) []string {
	participants := make([]string, 0, 4)
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		participants = append(participants, id)
	}
	for depth := 0; act != nil && depth < maxActChainScan; depth++ {
		actor, _ := act["sub"].(string)
		add(actor)
		nested, ok := act["act"].(map[string]any)
		if !ok {
			break
		}
		act = nested
	}
	add(sub)
	return participants
}

// actChainDelegates は act チェーンに subject と異なる行為者がいるかを返す。
// 自分自身だけを指す act は代行ではない。workload identity の自己交換は
// sub と act.sub が同じクライアントになるので、これを代行と数えると
// 自律実行が利用者の代理として記録されてしまう。
func actChainDelegates(act map[string]any, sub string) bool {
	for depth := 0; act != nil; depth++ {
		if depth >= maxActChainScan {
			return true
		}
		if actor, _ := act["sub"].(string); actor != "" && actor != sub {
			return true
		}
		nested, ok := act["act"].(map[string]any)
		if !ok {
			return false
		}
		act = nested
	}
	return false
}
