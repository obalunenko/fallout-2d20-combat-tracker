package domain

import (
	"strings"
	"time"
)

const CampaignDateLayout = "2006-01-02"

func ParseCampaignStartDate(value string) (time.Time, error) {
	return time.Parse(CampaignDateLayout, strings.TrimSpace(value))
}
