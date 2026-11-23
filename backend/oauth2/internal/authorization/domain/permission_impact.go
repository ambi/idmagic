package domain

import "errors"

// PermissionImpact は登録者の申告である。nil の PersonalData を false と区別する。
type PermissionImpact struct {
	Changes      string `json:"changes"`
	PersonalData *bool  `json:"personal_data"`
}

func (i *PermissionImpact) Validate() error {
	if i == nil {
		return nil
	}
	if i.PersonalData == nil {
		return errors.New("impact personal_data must be an explicit boolean")
	}
	switch i.Changes {
	case "none", "reversible", "irreversible":
		return nil
	}
	return errors.New("impact changes must be none, reversible or irreversible")
}

func (i *PermissionImpact) Clone() *PermissionImpact {
	if i == nil {
		return nil
	}
	cloned := *i
	if i.PersonalData != nil {
		value := *i.PersonalData
		cloned.PersonalData = &value
	}
	return &cloned
}
