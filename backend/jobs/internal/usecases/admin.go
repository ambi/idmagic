package usecases

// 管理者向けのジョブ参照と取り消し (wi-157, REQ-JOBS-012 / REQ-JOBS-013 / REQ-JOBS-014)。
//
// Jobs はキューへ仕事を積む HTTP の入口を持たない。ここが提供するのは参照と取り消しだけで、
// 再試行も再実行も強制完了も持たない。いずれも副作用をもう一度起こす操作であり、ハンドラーが
// 保証する冪等性の外側から引き金を引くことになるためである。

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrJobCursorMismatch はページングのカーソルが、それを発行したときと異なるテナントや
// 絞り込みとともに提示されたことを表す。黙って別の位置へ読み替えると、運用者は重複も
// 欠落も気づかないまま一覧を読むことになるので、続きではなく誤りとして返す。
var ErrJobCursorMismatch = errors.New("jobs: cursor was issued for a different tenant or filter")

// 一覧のページ既定値と上限 (SCL AdminJobQuery.limit)。
const (
	defaultAdminJobLimit = 50
	maxAdminJobLimit     = 200
)

// AdminJobDeps は管理系ユースケースの依存。Emit は取り消しの JobCanceled を配る。
type AdminJobDeps struct {
	Repo ports.JobRepository
	Emit func(spec.DomainEvent)
}

// TenantScope は呼び出し元の認可が確定させたテナントの範囲。呼び出し元 (ハンドラー) が
// ロールと経路から決めるものであり、リクエストのパラメーターからは決して作らない。
// AllTenants は system_admin が制御面テナントの経路で明示した場合にだけ立つ。
type TenantScope struct {
	TenantID   string
	AllTenants bool
}

func (s TenantScope) valid() bool { return s.AllTenants || s.TenantID != "" }

// includes は job がこの範囲から見えるかを返す。
func (s TenantScope) includes(job *domain.Job) bool {
	return s.AllTenants || (job != nil && job.TenantID == s.TenantID)
}

// ListJobsInput は一覧の入力。Scope 以外はすべて任意の絞り込みである。
type ListJobsInput struct {
	Scope    TenantScope
	Statuses []domain.JobStatus
	Kinds    []domain.JobKind
	Lane     domain.ExecutionLane
	Limit    int
	Cursor   string
}

// JobPage は 1 ページ分の結果。NextCursor が空なら最終ページである。
type JobPage struct {
	Jobs       []*domain.Job
	NextCursor string
}

// ListJobsForAdmin は 1 ページ分の Job を新しい順で返す。
//
// テナントの範囲は Scope が決め、リクエストのパラメーターは絞り込みにしか使わない。範囲を
// 与えられなかった入力は全件へ退避せずに拒否する。省略が全テナントの参照になる設計は、
// 一度の書き漏らしがテナント境界の穴になる。
func ListJobsForAdmin(ctx context.Context, deps AdminJobDeps, in ListJobsInput) (JobPage, error) {
	if !in.Scope.valid() {
		return JobPage{}, ports.ErrAdminJobFilterUnscoped
	}
	if deps.Repo == nil {
		return JobPage{}, errors.New("jobs: admin listing requires a repository")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultAdminJobLimit
	}
	if limit > maxAdminJobLimit {
		limit = maxAdminJobLimit
	}

	filter := ports.AdminJobFilter{
		TenantID:   in.Scope.TenantID,
		AllTenants: in.Scope.AllTenants,
		Statuses:   in.Statuses,
		Kinds:      in.Kinds,
		Lane:       in.Lane,
		// 1 件多く読むことで、次のページがあるかを別の COUNT なしに判定する。
		Limit: limit + 1,
	}
	if in.Cursor != "" {
		anchor, err := decodeJobCursor(in.Cursor)
		if err != nil {
			return JobPage{}, err
		}
		if anchor.Fingerprint != jobFilterFingerprint(in) {
			return JobPage{}, ErrJobCursorMismatch
		}
		filter.BeforeCreatedAt = anchor.CreatedAt
		filter.BeforeID = anchor.ID
	}

	jobs, err := deps.Repo.ListForAdmin(ctx, filter)
	if err != nil {
		return JobPage{}, err
	}
	page := JobPage{Jobs: jobs}
	if len(jobs) > limit {
		page.Jobs = jobs[:limit]
		last := page.Jobs[len(page.Jobs)-1]
		page.NextCursor = encodeJobCursor(jobCursor{
			CreatedAt: last.CreatedAt, ID: last.ID, Fingerprint: jobFilterFingerprint(in),
		})
	}
	return page, nil
}

// GetJobForAdmin は 1 件を返す。範囲の外にある Job は ErrJobNotFound とする。存在を
// 区別できる応答を返すと、id を総当たりするだけで他テナントに Job があることが分かる。
func GetJobForAdmin(ctx context.Context, deps AdminJobDeps, jobID string, scope TenantScope) (*domain.Job, error) {
	if !scope.valid() {
		return nil, ports.ErrAdminJobFilterUnscoped
	}
	if deps.Repo == nil {
		return nil, errors.New("jobs: admin read requires a repository")
	}
	job, err := deps.Repo.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if !scope.includes(job) {
		return nil, ports.ErrJobNotFound
	}
	return job, nil
}

// CancelJobForAdmin は終端に達していない Job を取り消す。
//
// 実行中のハンドラーはその場で中断されない。取り消しがリースを外すので、次のハートビートか
// 完了報告が失敗し、ハンドラー自身が処理をやめる。すでに確定した副作用は元へ戻らない。
// 終端に達した Job は成功として黙認せず拒否する。止めるよう頼んだ運用者にとって、すでに
// 終わっていたのか止まったのかは別の事実である。
func CancelJobForAdmin(ctx context.Context, deps AdminJobDeps, jobID string, scope TenantScope, now time.Time) (*domain.Job, error) {
	// 取り消しの前に範囲を確かめる。先に取り消してから範囲外と判定したのでは、他テナントの
	// Job を止めたうえで「存在しない」と答えることになる。
	if _, err := GetJobForAdmin(ctx, deps, jobID, scope); err != nil {
		return nil, err
	}
	canceled, err := deps.Repo.Cancel(ctx, jobID, now)
	if err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.JobCanceled{At: now, JobID: canceled.ID, TenantID: canceled.TenantID})
	return canceled, nil
}

// jobCursor はキーセットの位置と、それを発行したときの絞り込みの指紋。
type jobCursor struct {
	CreatedAt   time.Time
	ID          string
	Fingerprint string
}

// encodeJobCursor / decodeJobCursor はカーソルを不透明な 1 つの文字列にする。
//
// 署名は付けない。カーソルは権限ではなく位置しか運ばず、テナントの範囲はページごとに
// サーバー側で必ず付け直すからである。指紋は改竄への対策ではなく、条件を変えたまま続きを
// 取りにいった呼び出しを、静かな重複や欠落ではなく誤りとして返すためのものである。
func encodeJobCursor(c jobCursor) string {
	raw := strings.Join([]string{c.CreatedAt.UTC().Format(time.RFC3339Nano), c.ID, c.Fingerprint}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeJobCursor(s string) (jobCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return jobCursor{}, ErrJobCursorMismatch
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return jobCursor{}, ErrJobCursorMismatch
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return jobCursor{}, ErrJobCursorMismatch
	}
	return jobCursor{CreatedAt: createdAt, ID: parts[1], Fingerprint: parts[2]}, nil
}

// jobFilterFingerprint は、カーソルを発行したときのテナントの範囲と絞り込みを 1 つの
// 文字列にまとめる。並びは入力の順に依存するので、呼び出し側が値を並べ替えると別の
// 指紋になる。誤って続きとして扱うより、先頭から引き直させるほうが安全である。
func jobFilterFingerprint(in ListJobsInput) string {
	statuses := make([]string, len(in.Statuses))
	for i, s := range in.Statuses {
		statuses[i] = string(s)
	}
	kinds := make([]string, len(in.Kinds))
	for i, k := range in.Kinds {
		kinds[i] = string(k)
	}
	return fmt.Sprintf("%s/%t/%s/%s/%s",
		in.Scope.TenantID, in.Scope.AllTenants,
		strings.Join(statuses, ","), strings.Join(kinds, ","), in.Lane)
}
