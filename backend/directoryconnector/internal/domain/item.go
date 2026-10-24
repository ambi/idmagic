package domain

// Connector API のワイヤの形。TypeSpec の IdMagic.Sourcing の DirectoryUserItem、IngestionRunMode、
// DirectoryUserBatchRequest に従う。Connector はサーバーと別に配布するので、サーバーの内部の型を共有せず、
// 公開した契約だけに合わせる。

// Mode は取り込みの実行の種類である。
type Mode string

const (
	ModeFull        Mode = "full"
	ModeIncremental Mode = "incremental"
)

// MaxBatchItems は一つのバッチに入れられる項目の数の上限である。
const MaxBatchItems = 100

// Item は User の項目である。Connector は削除の項目を送らない。enabled は省くと true として扱われるので、
// false を落とさないよう常に送る。
type Item struct {
	ExternalID        string `json:"external_id"`
	Enabled           bool   `json:"enabled"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	Email             string `json:"email,omitempty"`
	EmployeeNumber    string `json:"employee_number,omitempty"`
	Department        string `json:"department,omitempty"`
}
