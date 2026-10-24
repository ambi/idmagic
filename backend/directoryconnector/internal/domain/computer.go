package domain

import (
	"encoding/base64"
	"encoding/binary"
	"strconv"
	"strings"
)

// ComputerItem は AD のコンピューターアカウントの項目である。TypeSpec の IdMagic.Sourcing の
// DirectoryComputerItem に従う。enabled は省くと true として扱われるので、常に送る。
type ComputerItem struct {
	ExternalID     string `json:"external_id"`
	Enabled        bool   `json:"enabled"`
	SAMAccountName string `json:"sam_account_name"`
	ObjectSID      string `json:"object_sid"`
	Domain         string `json:"domain"`
	DNSHostName    string `json:"dns_host_name,omitempty"`
}

// DefaultComputerFilter は computers.filter のデフォルト値である。
const DefaultComputerFilter = "(objectCategory=computer)"

const (
	adSAMAccountName = "sAMAccountName"
	adObjectSID      = "objectSid"
	adDNSHostName    = "dNSHostName"
	// dnDomainComponent は、識別名からドメインを導けないエントリに示す属性の名前である。
	dnDomainComponent = "DC"
)

// ComputerAttributes はコンピューターの検索で要求する属性である。
func ComputerAttributes() []string {
	return []string{adObjectGUID, adAccountControl, adSAMAccountName, adObjectSID, adDNSHostName}
}

// MapComputer は AD のエントリをコンピューターの項目へ写す (REQ-DIRECTORYCONNECTOR-007)。
func MapComputer(entry Entry) (ComputerItem, error) {
	missing := func(attribute string) (ComputerItem, error) {
		return ComputerItem{}, &IncompleteEntryError{DN: entry.DN, Attribute: attribute}
	}
	guid, ok := entry.first(adObjectGUID)
	if !ok || len(guid) != 16 {
		return missing(adObjectGUID)
	}
	sam, ok := entry.first(adSAMAccountName)
	if !ok || len(sam) == 0 {
		return missing(adSAMAccountName)
	}
	rawSID, _ := entry.first(adObjectSID)
	sid, ok := FormatSID(rawSID)
	if !ok {
		return missing(adObjectSID)
	}
	control, ok := entry.first(adAccountControl)
	flags, err := strconv.ParseInt(string(control), 10, 64)
	if !ok || err != nil {
		return missing(adAccountControl)
	}
	domain := DomainFromDN(entry.DN)
	if domain == "" {
		return missing(dnDomainComponent)
	}
	host, _ := entry.first(adDNSHostName)
	return ComputerItem{
		ExternalID: base64.StdEncoding.EncodeToString(guid), Enabled: flags&adAccountDisabledBit == 0,
		SAMAccountName: string(sam), ObjectSID: sid, Domain: domain, DNSHostName: string(host),
	}, nil
}

// FormatSID は objectSid の二進表現を S-<revision>-<authority>-<副権限>... の文字列にする。identifier
// authority は 6 バイトのビッグエンディアン、副権限は 4 バイトのリトルエンディアンである (MS-DTYP 2.4.2.2)。
// 副権限のない値と、長さが副権限の数と合わない値は読めないものとする。
func FormatSID(raw []byte) (string, bool) {
	if len(raw) < 8 || raw[1] == 0 || len(raw) != 8+4*int(raw[1]) {
		return "", false
	}
	var authority uint64
	for _, b := range raw[2:8] {
		authority = authority<<8 | uint64(b)
	}
	var sid strings.Builder
	sid.WriteString("S-" + strconv.Itoa(int(raw[0])) + "-" + strconv.FormatUint(authority, 10))
	for offset := 8; offset < len(raw); offset += 4 {
		sid.WriteString("-" + strconv.FormatUint(uint64(binary.LittleEndian.Uint32(raw[offset:])), 10))
	}
	return sid.String(), true
}

// DomainFromDN は識別名の DC= の値を現れた順に . でつなぎ、小文字にする。DC= がなければ空を返す。
func DomainFromDN(dn string) string {
	var labels []string
	for rdn := range strings.SplitSeq(dn, ",") {
		name, value, ok := strings.Cut(strings.TrimSpace(rdn), "=")
		if ok && strings.EqualFold(name, dnDomainComponent) && value != "" {
			labels = append(labels, strings.ToLower(value))
		}
	}
	return strings.Join(labels, ".")
}
