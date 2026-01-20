package credits

import (
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

type OrganizationCreditDetailsResult struct {
	Meta *responsemeta.ResponseMeta
	Data OrganizationCreditDetails
}

type OrganizationCreditDetails struct {
	Balance             int64               `json:"balance"`
	CreditExpirations   []CreditExpiration  `json:"credit_expirations"`
	AutoReplenishConfig AutoReplenishConfig `json:"auto_replenish_config"`
}

type CreditExpiration struct {
	Balance        int64                `json:"balance"`
	CreationDate   mo.Option[time.Time] `json:"creation_date,omitzero"`
	ExpirationDate mo.Option[time.Time] `json:"expiration_date,omitzero"`
}

type AutoReplenishConfig struct {
	Enabled   bool             `json:"enabled"`
	Threshold mo.Option[int64] `json:"threshold,omitzero"`
	Amount    mo.Option[int64] `json:"amount,omitzero"`
}

func parseOrganizationCreditDetails(credits *components.OrganizationCredits) OrganizationCreditDetails {
	autoReplenishConfig := AutoReplenishConfig{
		Enabled: credits.AutoReplenishConfig.Enabled,
	}
	if credits.GetAutoReplenishConfig().Threshold != nil {
		autoReplenishConfig.Threshold = mo.Some(*credits.GetAutoReplenishConfig().Threshold)
	}
	if credits.GetAutoReplenishConfig().Amount != nil {
		autoReplenishConfig.Amount = mo.Some(*credits.GetAutoReplenishConfig().Amount)
	}
	var creditExpirations []CreditExpiration
	if len(credits.GetCreditExpirations()) > 0 {
		creditExpirations = make([]CreditExpiration, 0, len(credits.GetCreditExpirations()))
		for _, creditExpiration := range credits.GetCreditExpirations() {
			ce := CreditExpiration{
				Balance: creditExpiration.Balance,
			}
			if creditExpiration.CreatedAt != nil {
				ce.CreationDate = mo.Some(*creditExpiration.CreatedAt)
			}
			if creditExpiration.ExpiresAt != nil {
				ce.ExpirationDate = mo.Some(*creditExpiration.ExpiresAt)
			}
			creditExpirations = append(creditExpirations, ce)
		}
	}
	return OrganizationCreditDetails{
		Balance:             credits.Balance,
		CreditExpirations:   creditExpirations,
		AutoReplenishConfig: autoReplenishConfig,
	}
}

type UserCreditDetailsResult struct {
	Meta *responsemeta.ResponseMeta
	Data UserCreditDetails
}

type UserCreditDetails struct {
	Balance  int64                `json:"balance"`
	ResetsAt mo.Option[time.Time] `json:"resets_at,omitzero"`
}

func parseUserCreditDetails(credits *components.UserCredits) UserCreditDetails {
	ucd := UserCreditDetails{
		Balance: credits.Balance,
	}
	if credits.ResetsAt != nil {
		ucd.ResetsAt = mo.Some(*credits.ResetsAt)
	}
	return ucd
}
