package database

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"z42-core/internal/types"
)

type ObjectId string

const EmptyObjectId ObjectId = ""

func NewObjectId() ObjectId {
	return ObjectId(uuid.New().String())
}

type UserStatus string

type User struct {
	Id       ObjectId
	Email    string
	Password string
	Status   UserStatus
}

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusPending  UserStatus = "pending"
)

type UserRole string

const (
	UserRoleOwner    UserRole = "owner"
	UserRoleRead     UserRole = "read"
	UserRoleWrite    UserRole = "write"
	UserRoleDisabled UserRole = "disabled"
)

type NewUser struct {
	Email    string
	Password string
	Status   UserStatus
}

type VerificationType string

const (
	VerificationTypeSignup  VerificationType = "signup"
	VerificationTypeRecover VerificationType = "recover"
)

type Verification struct {
	Code string
	Type VerificationType
}

type Zone struct {
	Id              ObjectId
	Name            string
	Enabled         bool
	Dnssec          bool
	CNameFlattening bool
	SOA             types.SOA_RRSet
	DS              string
}

type NewZone struct {
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	Dnssec          bool            `json:"dnssec"`
	CNameFlattening bool            `json:"cname_flattening"`
	SOA             types.SOA_RRSet `json:"soa"`
	Keys            types.ZoneKeys  `json:"keys"`
	NS              types.NS_RRSet  `json:"ns"`
}

type ZoneUpdate struct {
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	Dnssec          bool            `json:"dnssec"`
	CNameFlattening bool            `json:"cname_flattening"`
	SOA             types.SOA_RRSet `json:"soa"`
}

type ZoneDelete struct {
	Name string `json:"name"`
}

type ZoneImport struct {
	Name    string                            `json:"name"`
	Entries map[string]map[string]types.RRSet `json:"entries"`
}

func (zi *ZoneImport) UnmarshalJSON(data []byte) error {
	var _zi struct {
		Name    string                                `json:"name"`
		Entries map[string]map[string]json.RawMessage `json:"entries"`
	}
	err := jsoniter.Unmarshal(data, &_zi)
	if err != nil {
		return err
	}
	zi.Name = _zi.Name
	zi.Entries = make(map[string]map[string]types.RRSet)
	for label, location := range _zi.Entries {
		zi.Entries[label] = make(map[string]types.RRSet)
		for rtype, rvalue := range location {
			rrset := types.TypeStrToRRSet(rtype)
			if err = jsoniter.Unmarshal(rvalue, rrset); err != nil {
				return err
			}
			zi.Entries[label][rtype] = rrset
		}
	}
	return nil
}

type Location struct {
	Id      ObjectId
	Name    string
	Enabled bool
}

type NewLocation struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
	Enabled  bool   `json:"enabled"`
}

type LocationUpdate struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
	Enabled  bool   `json:"enabled"`
}

type LocationDelete struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
}

type RecordSet struct {
	Id      ObjectId
	Type    string
	Value   types.RRSet
	Enabled bool
}

type NewRecordSet struct {
	ZoneName string      `json:"zone_name"`
	Location string      `json:"location"`
	Type     string      `json:"type"`
	Value    types.RRSet `json:"value"`
	Enabled  bool        `json:"enabled"`
}

func (r *NewRecordSet) UnmarshalJSON(data []byte) error {
	var dat struct {
		ZoneName string `json:"zone_name"`
		Location string `json:"location"`
		Type     string `json:"type"`
		Enabled  bool   `json:"enabled"`
	}
	if err := jsoniter.Unmarshal(data, &dat); err != nil {
		return err
	}
	value := types.TypeStrToRRSet(dat.Type)
	if value == nil {
		return errors.New("invalid record type")
	}
	val := struct {
		Value types.RRSet `json:"value"`
	}{
		Value: value,
	}
	if err := jsoniter.Unmarshal(data, &val); err != nil {
		return err
	}
	r.ZoneName = dat.ZoneName
	r.Location = dat.Location
	r.Type = dat.Type
	r.Enabled = dat.Enabled
	r.Value = val.Value
	return nil
}

type RecordSetUpdate struct {
	ZoneName string      `json:"zone_name"`
	Location string      `json:"location"`
	Type     string      `json:"type"`
	Value    types.RRSet `json:"value"`
	Enabled  bool        `json:"enabled"`
}

func (r *RecordSetUpdate) UnmarshalJSON(data []byte) error {
	var dat struct {
		ZoneName string `json:"zone_name"`
		Location string `json:"location"`
		Type     string `json:"type"`
		Enabled  bool   `json:"enabled"`
	}
	if err := jsoniter.Unmarshal(data, &dat); err != nil {
		return err
	}
	value := types.TypeStrToRRSet(dat.Type)
	if value == nil {
		return errors.New("invalid record type")
	}
	val := struct {
		Value types.RRSet `json:"value"`
	}{
		Value: value,
	}
	if err := jsoniter.Unmarshal(data, &val); err != nil {
		return err
	}
	r.ZoneName = dat.ZoneName
	r.Location = dat.Location
	r.Type = dat.Type
	r.Enabled = dat.Enabled
	r.Value = val.Value
	return nil
}

type RecordSetDelete struct {
	ZoneName string `json:"zone_name"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

type ListItem struct {
	Id      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type List struct {
	Items []ListItem `json:"items"`
	Total int        `json:"total"`
}

func EmptyList() List {
	return List{
		Items: []ListItem{},
		Total: 0,
	}
}

type Event struct {
	Revision int
	ZoneId   string
	Type     EventType
	Value    string
}

type EventType string

const (
	AddZone        EventType = "add_zone"
	UpdateZone     EventType = "update_zone"
	DeleteZone     EventType = "delete_zone"
	ImportZone     EventType = "import_zone"
	AddLocation    EventType = "add_location"
	UpdateLocation EventType = "update_location"
	DeleteLocation EventType = "delete_location"
	AddRecord      EventType = "add_record"
	UpdateRecord   EventType = "update_record"
	DeleteRecord   EventType = "delete_record"
)

type APIKey struct {
	Name    string
	Scope   APIKeyScope
	Hash    string
	Enabled bool
	ZoneId  ObjectId
	UserId  ObjectId
}

type APIKeyScope string

const (
	ACME APIKeyScope = "acme"
)

type APIKeyItem struct {
	Name     string `json:"name"`
	Scope    string `json:"scope"`
	ZoneName string `json:"zone_name"`
	Enabled  bool   `json:"enabled"`
}

type APIKeyUpdate struct {
	Name    string `json:"name"`
	Scope   string `json:"scope"`
	Enabled bool   `json:"enabled"`
}
