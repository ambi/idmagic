package domain

import (
	"slices"
	"testing"
)

// 名指しで返ることを固定する。bool ではなく語彙の外の値そのものを返すのは、拒否した側が
// 何を拒否したのかを言えないと、要素を足した人にとって手がかりにならないからである。
//
//spec:covers RFC8176-AMR-VOCABULARY: 語彙が閉じていること、つまり宣言された値だけが通り、それ以外は
func TestUnknownAMRValuesNamesWhatIsOutsideTheVocabulary(t *testing.T) {
	t.Parallel()

	for _, value := range AMRVocabulary() {
		if got := UnknownAMRValues([]string{value}); len(got) != 0 {
			t.Fatalf("declared value %q reported as unknown: %v", value, got)
		}
	}
	// RFC 8176 の登録値であっても、この製品が宣言していなければ語彙の外である。
	got := UnknownAMRValues([]string{"pwd", "mfa", "rc", "pop"})
	if !slices.Equal(got, []string{"mfa", "pop"}) {
		t.Fatalf("unknown=%v, want [mfa pop]", got)
	}
	if got := UnknownAMRValues(nil); len(got) != 0 {
		t.Fatalf("nil amr reported %v", got)
	}
}

// 語彙に無い値が MFA 充足として通ると、語彙の検査をすり抜けた値が強度の申告まで届く。
//
//spec:covers RFC8176-AMR-VOCABULARY: acr を mfa へ上げる部分集合が語彙の内側にあることを固定する。
func TestMfaAMRValuesAreASubsetOfTheVocabulary(t *testing.T) {
	t.Parallel()

	for _, value := range MfaAMRValues() {
		if !slices.Contains(AMRVocabulary(), value) {
			t.Fatalf("%q raises acr but is not in the vocabulary", value)
		}
	}
	// federated は語彙にあるが MFA 充足ではない。上流の IdP が何を検証したかは分からない。
	if slices.Contains(MfaAMRValues(), AMRFederated) {
		t.Fatal("federated must not count as a second factor")
	}
	// pwd も同様に MFA 充足ではない。
	if slices.Contains(MfaAMRValues(), AMRPassword) {
		t.Fatal("pwd must not count as a second factor")
	}
}
