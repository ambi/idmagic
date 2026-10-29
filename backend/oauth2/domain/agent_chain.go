package domain

// MergeAgentChain は、発行時に権限を支えた Agent の識別子を、最初に現れた順のまま重複と
// 空文字列を除いて 1 つに並べる (REQ-OAUTH2-012)。トークン交換は、要求したクライアントの
// Agent、subject_token と actor_token の Agent、ワークロード ID 連携の Agent をここで束ね、
// 交換を重ねても連鎖の途中の Agent を落とさない。
func MergeAgentChain(groups ...[]string) []string {
	var merged []string
	seen := map[string]bool{}
	for _, group := range groups {
		for _, id := range group {
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			merged = append(merged, id)
		}
	}
	return merged
}
