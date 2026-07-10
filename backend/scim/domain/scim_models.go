package domain

type ScimErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	Detail   string   `json:"detail,omitempty"`
	ScimType string   `json:"scimType,omitempty"`
}

func NewScimError(status, detail, scimType string) ScimErrorResponse {
	return ScimErrorResponse{
		Schemas:  []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Status:   status,
		Detail:   detail,
		ScimType: scimType,
	}
}

type ListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    []any    `json:"Resources"`
}

type Name struct {
	Formatted  string `json:"formatted,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
}

type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type UserResource struct {
	Schemas  []string `json:"schemas"`
	ID       string   `json:"id"`
	UserName string   `json:"userName"`
	Name     *Name    `json:"name,omitempty"`
	Emails   []Email  `json:"emails,omitempty"`
	Active   bool     `json:"active"`
	Meta     Meta     `json:"meta"`
}

type Meta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
}

type GroupMember struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
}

type GroupResource struct {
	Schemas     []string      `json:"schemas"`
	ID          string        `json:"id"`
	DisplayName string        `json:"displayName"`
	Members     []GroupMember `json:"members"`
	Meta        Meta          `json:"meta"`
}

// ServiceProviderConfig
type BulkConfig struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

type FilterConfig struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

type ServiceProviderConfig struct {
	Schemas          []string `json:"schemas"`
	DocumentationUri string   `json:"documentationUri,omitempty"`
	Patch            struct {
		Supported bool `json:"supported"`
	} `json:"patch"`
	Bulk           BulkConfig   `json:"bulk"`
	Filter         FilterConfig `json:"filter"`
	ChangePassword struct {
		Supported bool `json:"supported"`
	} `json:"changePassword"`
	Sort struct {
		Supported bool `json:"supported"`
	} `json:"sort"`
	Etag struct {
		Supported bool `json:"supported"`
	} `json:"etag"`
	AuthenticationSchemes []AuthenticationScheme `json:"authenticationSchemes"`
}

// AuthenticationScheme は ServiceProviderConfig が申告する認証方式 (RFC 7643 §5)。
type AuthenticationScheme struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SpecUri     string `json:"specUri,omitempty"`
	Type        string `json:"type"`
}

type ResourceType struct {
	Schemas     []string `json:"schemas"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Endpoint    string   `json:"endpoint"`
	Description string   `json:"description,omitempty"`
	Schema      string   `json:"schema"`
}

type SchemaAttribute struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	MultiValued   bool              `json:"multiValued"`
	Description   string            `json:"description,omitempty"`
	Required      bool              `json:"required"`
	CaseExact     bool              `json:"caseExact"`
	Mutability    string            `json:"mutability"`
	Returned      string            `json:"returned"`
	Uniqueness    string            `json:"uniqueness"`
	SubAttributes []SchemaAttribute `json:"subAttributes,omitempty"`
}

type Schema struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Attributes  []SchemaAttribute `json:"attributes"`
}

// PATCH Operations
type PatchOp struct {
	Schemas    []string    `json:"schemas"`
	Operations []Operation `json:"Operations"`
}

type Operation struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}
