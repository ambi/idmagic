package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/sourceuser"
)

func (u *Usecases) CreateUser(ctx context.Context, tenantID string, body map[string]any) (map[string]any, error) {
	w, err := domain.ParseUserWrite(body)
	if err != nil {
		return nil, err
	}

	existing, err := u.UserRepo.FindByUsername(ctx, tenantID, w.UserName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: userName %q already exists", ErrDuplicate, w.UserName)
	}

	scimIDBytes := make([]byte, 16)
	if _, err := rand.Read(scimIDBytes); err != nil {
		return nil, err
	}
	scimID := hex.EncodeToString(scimIDBytes)

	attributes, err := u.sourceAttributes(ctx, tenantID, w)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user := &userdomain.User{
		TenantID:      tenantID,
		PasswordHash:  "", // SCIM users usually don't have local password initially
		EmailVerified: true,
		Roles:         []string{},
		Lifecycle: userdomain.UserLifecycle{
			Status:          userStatusFromActive(w.Active),
			StatusChangedAt: &now,
		},
		Attributes: make(map[string]userdomain.AttributeValue),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	attributes.Apply(user)

	user, err = u.Writes.CreateUser(ctx, scimID, user)
	if err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

func userStatusFromActive(active bool) idmdomain.UserStatus {
	if active {
		return idmdomain.UserStatusActive
	}
	return idmdomain.UserStatusDisabled
}

func (u *Usecases) GetUser(ctx context.Context, tenantID, scimID string) (map[string]any, error) {
	user, err := u.findLiveUserByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

// UpdateUser は PUT の省略属性を既定値へ戻し、属性を所有者の操作へ渡す。
// 本文の検証と参照の解決は、書き込みの前に終える。
func (u *Usecases) UpdateUser(ctx context.Context, tenantID, scimID string, body map[string]any) (map[string]any, error) {
	user, err := u.findLiveUserByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	user = editableCopy(user)

	w, err := domain.ParseUserWrite(body)
	if err != nil {
		return nil, err
	}
	if w.UserName != user.PreferredUsername {
		if existing, err := u.UserRepo.FindByUsername(ctx, tenantID, w.UserName); err != nil {
			return nil, err
		} else if existing != nil && existing.ID != user.ID {
			return nil, fmt.Errorf("%w: userName %q already exists", ErrDuplicate, w.UserName)
		}
	}

	attributes, err := u.sourceAttributes(ctx, tenantID, w)
	if err != nil {
		return nil, err
	}
	attributes.Apply(user)
	user.UpdatedAt = time.Now()

	user, err = u.Writes.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	user, err = u.applyActive(ctx, tenantID, user, w.Active)
	if err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

// PatchUser は RFC 7644 §3.5.2 の操作を検証し、投影した属性を所有者の操作へ渡す。
// active の変更は、属性の保存後に既存のライフサイクル操作へ渡す。
func (u *Usecases) PatchUser(ctx context.Context, tenantID, scimID string, body map[string]any) (map[string]any, error) {
	user, err := u.findLiveUserByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	user = editableCopy(user)

	ops, err := domain.ParseUserPatchOps(body)
	if err != nil {
		return nil, err
	}

	active := user.Lifecycle.Status == idmdomain.UserStatusActive
	for _, op := range ops {
		if op.Attr == domain.UserAttrActive {
			active = patchActiveValue(op)
			continue
		}
		if err := u.applyUserPatchOp(ctx, tenantID, user, op); err != nil {
			return nil, err
		}
	}

	user.UpdatedAt = time.Now()
	user, err = u.Writes.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	user, err = u.applyActive(ctx, tenantID, user, active)
	if err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

func (u *Usecases) applyUserPatchOp(ctx context.Context, tenantID string, user *userdomain.User, op domain.UserPatchOp) error {
	isRemoveOp := op.Op == "remove"

	switch op.Attr {
	case domain.UserAttrUserName:
		if isRemoveOp {
			return domain.NewMutationError("invalidValue", "userName cannot be removed")
		}
		userName, _ := op.Value.(string)
		if userName == "" {
			return domain.NewMutationError("invalidValue", "userName value must be a non-empty string")
		}
		if userName != user.PreferredUsername {
			existing, err := u.UserRepo.FindByUsername(ctx, tenantID, userName)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != user.ID {
				return fmt.Errorf("%w: userName %q already exists", ErrDuplicate, userName)
			}
		}
		user.PreferredUsername = userName
	case domain.UserAttrName:
		if isRemoveOp {
			empty := ""
			user.GivenName, user.FamilyName, user.Name = &empty, &empty, &empty
			return nil
		}
		nameMap, ok := op.Value.(map[string]any)
		if !ok {
			return domain.NewMutationError("invalidValue", "name value must be an object")
		}
		givenName, _ := nameMap["givenName"].(string)
		familyName, _ := nameMap["familyName"].(string)
		formatted, _ := nameMap["formatted"].(string)
		user.GivenName, user.FamilyName, user.Name = &givenName, &familyName, &formatted
	case domain.UserAttrGivenName:
		user.GivenName = patchStringField(op)
	case domain.UserAttrFamilyName:
		user.FamilyName = patchStringField(op)
	case domain.UserAttrFormatted:
		user.Name = patchStringField(op)
	case domain.UserAttrEmails:
		if isRemoveOp {
			empty := ""
			user.Email = &empty
			return nil
		}
		email, _ := op.Value.(string)
		user.Email = &email
	case domain.UserAttrEmployeeNumber:
		sourceuser.SetString(user, sourceuser.AttrEmployeeNumber, *patchStringField(op))
	case domain.UserAttrDepartment:
		sourceuser.SetString(user, sourceuser.AttrDepartment, *patchStringField(op))
	case domain.UserAttrManager:
		managerSub, err := u.resolveManagerSub(ctx, tenantID, *patchStringField(op))
		if err != nil {
			return err
		}
		sourceuser.SetString(user, sourceuser.AttrManagerSub, managerSub)
	}
	return nil
}

// editableCopy は保存先が返した User を、書き込みの途中で失敗しても保存先へ届かない
// 複製にする。リポジトリによっては返す値が保存済みの値そのものなので、Save を呼ばない
// 失敗でも書き換えが残る。書き込みがその場で変える参照型の field は Attributes だけで、
// ポインターの field は差し替えるので浅い複製で足りる。
func editableCopy(user *userdomain.User) *userdomain.User {
	clone := *user
	clone.Attributes = maps.Clone(user.Attributes)
	return &clone
}

// resolveManagerSub resolves an enterprise extension "manager" SCIM id to
// the manager's internal User.sub, scoped to tenantID so a reference can
// never cross a tenant boundary (wi-247 Risk Notes). An empty
// managerScimID (manager omitted/cleared) resolves to "". An unresolvable
// id is a *domain.MutationError (invalidValue), returned before any
// persistence write (validate-first).
func (u *Usecases) resolveManagerSub(ctx context.Context, tenantID, managerScimID string) (string, error) {
	if strings.TrimSpace(managerScimID) == "" {
		return "", nil
	}
	return u.resolveLiveUserID(ctx, tenantID, "manager", managerScimID)
}

// sourceAttributes は POST と PUT の属性を、manager を解決した取り込み元の属性へ写す。
// w の空の値は属性を消すことを表す (PUT の完全な置換)。
func (u *Usecases) sourceAttributes(ctx context.Context, tenantID string, w domain.UserWrite) (sourceuser.Attributes, error) {
	managerSub, err := u.resolveManagerSub(ctx, tenantID, w.ManagerValue)
	if err != nil {
		return sourceuser.Attributes{}, err
	}
	return sourceuser.Attributes{
		PreferredUsername: w.UserName, Name: w.Formatted, GivenName: w.GivenName, FamilyName: w.FamilyName, Email: w.Email,
		EmployeeNumber: w.EmployeeNumber, Department: w.Department, ManagerSub: &managerSub,
	}, nil
}

func patchStringField(op domain.UserPatchOp) *string {
	if op.Op == "remove" {
		empty := ""
		return &empty
	}
	s, _ := op.Value.(string)
	return &s
}

// patchActiveValue は active への PATCH の操作が求める値を返す。remove は既定値の true に戻す。
func patchActiveValue(op domain.UserPatchOp) bool {
	if op.Op == "remove" {
		return true
	}
	active, _ := op.Value.(bool)
	return active
}

// applyActive は、保存済みの User の active を求める値へ合わせ、応答へ射影する User を返す。
// 無効化と再有効化は IdManagement の操作へ渡し、User の状態を直接保存しない。止めることに伴う
// イベント、端末の失効、所有する Agent の無効化を、管理 API で止めたときと揃えるためである。
func (u *Usecases) applyActive(ctx context.Context, tenantID string, user *userdomain.User, active bool) (*userdomain.User, error) {
	if active == (user.Lifecycle.Status == idmdomain.UserStatusActive) {
		return user, nil
	}
	if _, err := u.UserLifecycle.SetUserDisabled(ctx, tenantID, user.ID, !active, time.Now()); err != nil {
		return nil, err
	}
	updated, err := u.UserRepo.FindBySub(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrNotFound
	}
	return updated, nil
}

func (u *Usecases) DeleteUser(ctx context.Context, tenantID, scimID string) error {
	// 二重の削除も 404 にする (RFC7644-DELETE-SEMANTICS)。
	user, err := u.findLiveUserByScimID(ctx, tenantID, scimID)
	if err != nil {
		return err
	}

	// 削除は予約にとどめ、IdManagement の操作へ渡す。User は PendingDeletion になる。
	return u.UserLifecycle.ScheduleUserDeletion(ctx, tenantID, user.ID, time.Now())
}

func (u *Usecases) ListUsers(ctx context.Context, tenantID string, query ListQuery) (ListResult, error) {
	users, err := u.UserRepo.FindAll(ctx, tenantID)
	if err != nil {
		return ListResult{}, err
	}

	var expr domain.FilterExpr
	if query.Filter != "" {
		expr, err = domain.ParseFilter(query.Filter, domain.UserFilterAttributes)
		if err != nil {
			return ListResult{}, err
		}
	}

	var matched []map[string]any
	for _, user := range users {
		// 削除済みは Resources にも totalResults にも入れない
		// (RFC7644-DELETE-SEMANTICS)。フィルターより前に落とすので、
		// filter が何であれ現れない。
		if scimDeleted(user) {
			continue
		}

		ref, err := u.ScimRepo.FindUserRefByUserID(ctx, tenantID, user.ID)
		if err != nil {
			return ListResult{}, err
		}
		scimID := user.ID
		if ref != nil {
			scimID = ref.ScimID
		}

		if expr != nil && !expr.Matches(userFilterAttrs(user, scimID)) {
			continue
		}

		scimUser, err := u.toScimUser(ctx, tenantID, user, scimID)
		if err != nil {
			return ListResult{}, err
		}
		matched = append(matched, scimUser)
	}

	return paginate(matched, query)
}

// userFilterAttrs flattens a User into the lower-cased attribute map
// domain.UserFilterAttributes expects.
func userFilterAttrs(user *userdomain.User, scimID string) map[string]any {
	attrs := map[string]any{
		"username":          user.PreferredUsername,
		"active":            user.Lifecycle.Status == idmdomain.UserStatusActive,
		"id":                scimID,
		"meta.created":      user.CreatedAt.Format(time.RFC3339),
		"meta.lastmodified": user.UpdatedAt.Format(time.RFC3339),
	}
	if user.Name != nil {
		attrs["name.formatted"] = *user.Name
	}
	if user.GivenName != nil {
		attrs["name.givenname"] = *user.GivenName
	}
	if user.FamilyName != nil {
		attrs["name.familyname"] = *user.FamilyName
	}
	if user.Email != nil {
		attrs["emails.value"] = *user.Email
	}
	return attrs
}

// toScimUser projects a User onto its SCIM representation. The enterprise
// extension object (and its URN in schemas) is included only when the User
// carries at least one employee_number/department/manager_sub Attributes
// entry (RFC7643-ENTERPRISE-EXTENSION adoption:partial), matching how emails
// is already omitted when there is no canonical email.
func (u *Usecases) toScimUser(ctx context.Context, tenantID string, user *userdomain.User, scimID string) (map[string]any, error) {
	var givenName, familyName, formattedName string
	if user.GivenName != nil {
		givenName = *user.GivenName
	}
	if user.FamilyName != nil {
		familyName = *user.FamilyName
	}
	if user.Name != nil {
		formattedName = *user.Name
	}

	active := user.Lifecycle.Status == idmdomain.UserStatusActive

	schemas := []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	ext := map[string]any{}
	if v, ok := sourceuser.String(user.Attributes, sourceuser.AttrEmployeeNumber); ok {
		ext["employeeNumber"] = v
	}
	if v, ok := sourceuser.String(user.Attributes, sourceuser.AttrDepartment); ok {
		ext["department"] = v
	}
	if managerSub, ok := sourceuser.String(user.Attributes, sourceuser.AttrManagerSub); ok {
		// 削除済みの manager は出さない (RFC7644-DELETE-SEMANTICS)。書き込みが
		// 拒否する参照を読み取りが返すと、読んだ resource をそのまま送り返す
		// read-modify-write が 400 になる。
		manager, err := u.liveUserByID(ctx, managerSub)
		if err != nil {
			return nil, err
		}
		if manager != nil {
			managerRef, err := u.ScimRepo.FindUserRefByUserID(ctx, tenantID, managerSub)
			if err != nil {
				return nil, err
			}
			if managerRef != nil {
				ext["manager"] = map[string]any{"value": managerRef.ScimID}
			}
		}
	}
	if len(ext) > 0 {
		schemas = append(schemas, domain.EnterpriseUserSchemaURN)
	}

	resource := map[string]any{
		"schemas":  schemas,
		"id":       scimID,
		"userName": user.PreferredUsername,
		"name": map[string]any{
			"familyName": familyName,
			"givenName":  givenName,
			"formatted":  formattedName,
		},
		"active": active,
		"meta": map[string]any{
			"resourceType": "User",
			"created":      user.CreatedAt.Format(time.RFC3339),
			"lastModified": user.UpdatedAt.Format(time.RFC3339),
			"location":     "/scim/v2/Users/" + scimID,
		},
	}
	if user.Email != nil && strings.TrimSpace(*user.Email) != "" {
		resource["emails"] = []map[string]any{{
			"value":   *user.Email,
			"type":    "work",
			"primary": true,
		}}
	}
	if len(ext) > 0 {
		resource[domain.EnterpriseUserSchemaURN] = ext
	}
	return resource, nil
}
