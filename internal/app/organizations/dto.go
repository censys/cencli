package organizations

import (
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/google/uuid"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

type OrganizationDetailsResult struct {
	Meta *responsemeta.ResponseMeta
	Data OrganizationDetails
}

type OrganizationDetails struct {
	ID           uuid.UUID                           `json:"id"`
	CreatedAt    mo.Option[time.Time]                `json:"created_at,omitzero"`
	Name         string                              `json:"name"`
	MemberCounts *components.MemberCounts            `json:"member_counts,omitempty"`
	Preferences  *components.OrganizationPreferences `json:"preferences,omitempty"`
}

func parseOrganizationDetails(details *components.OrganizationDetails) OrganizationDetails {
	var id uuid.UUID = uuid.Nil
	if uid, err := uuid.Parse(details.UID); err == nil {
		id = uid
	}
	var createdAt mo.Option[time.Time]
	if details.CreatedAt != nil {
		createdAt = mo.Some(*details.CreatedAt)
	}
	return OrganizationDetails{
		ID:           id,
		CreatedAt:    createdAt,
		Name:         details.Name,
		MemberCounts: details.MemberCounts,
		Preferences:  details.Preferences,
	}
}

type OrganizationMembersResult struct {
	Meta *responsemeta.ResponseMeta
	Data OrganizationMembers
}

type OrganizationMembers struct {
	Members []OrganizationMember `json:"members"`
}

type OrganizationMember struct {
	ID              uuid.UUID            `json:"id"`
	CreatedAt       mo.Option[time.Time] `json:"created_at,omitzero"`
	Email           mo.Option[string]    `json:"email,omitzero"`
	FirstName       mo.Option[string]    `json:"first_name,omitzero"`
	LastName        mo.Option[string]    `json:"last_name,omitzero"`
	Roles           []string             `json:"roles,omitempty"`
	LatestLoginTime mo.Option[time.Time] `json:"latest_login_time,omitzero"`
	FirstLoginTime  mo.Option[time.Time] `json:"first_login_time,omitzero"`
}

func parseOrganizationMembers(members *components.OrganizationMembersList) OrganizationMembers {
	om := OrganizationMembers{
		Members: make([]OrganizationMember, 0, len(members.Members)),
	}
	for _, member := range members.Members {
		om.Members = append(om.Members, parseOrganizationMember(member))
	}
	return om
}

func parseOrganizationMember(member components.OrganizationMember) OrganizationMember {
	var id uuid.UUID = uuid.Nil
	if uid, err := uuid.Parse(member.UID); err == nil {
		id = uid
	}
	var createdAt mo.Option[time.Time]
	if member.CreatedAt != nil {
		createdAt = mo.Some(*member.CreatedAt)
	}
	var email mo.Option[string]
	if member.Email != "" {
		email = mo.Some(member.Email)
	}
	var firstName mo.Option[string]
	if member.FirstName != "" {
		firstName = mo.Some(member.FirstName)
	}
	var lastName mo.Option[string]
	if member.LastName != "" {
		lastName = mo.Some(member.LastName)
	}
	var latestLoginTime mo.Option[time.Time]
	if member.LatestLoginTime != nil {
		latestLoginTime = mo.Some(*member.LatestLoginTime)
	}
	var firstLoginTime mo.Option[time.Time]
	if member.FirstLoginTime != nil {
		firstLoginTime = mo.Some(*member.FirstLoginTime)
	}
	return OrganizationMember{
		ID:              id,
		CreatedAt:       createdAt,
		Email:           email,
		FirstName:       firstName,
		LastName:        lastName,
		Roles:           member.Roles,
		LatestLoginTime: latestLoginTime,
		FirstLoginTime:  firstLoginTime,
	}
}
