package model

import "time"

var AllowedModerationStatuses = []string{
	ModerationStatusCreated, ModerationStatusAssigned, ModerationStatusOnModeration,
	ModerationStatusChangesRequested, ModerationStatusFinished,
}
var AllowedModerationTypes = []string{
	ModerationTypeProfileChange, ModerationTypeProfileReport, ModerationTypeCharacterReport,
	ModerationTypeCampaignReport, ModerationTypeMessageReport,
}

const (
	ModerationTypeProfileChange   = "profileChange"
	ModerationTypeProfileReport   = "profileReport"
	ModerationTypeCharacterReport = "characterReport"
	ModerationTypeCampaignReport  = "campaignReport"
	ModerationTypeMessageReport   = "messageReport"

	ModerationStatusCreated          = "Created"
	ModerationStatusAssigned         = "Assigned"
	ModerationStatusOnModeration     = "OnModeration"
	ModerationStatusChangesRequested = "ChangesRequested"
	ModerationStatusFinished         = "Finished"
)

type ModerationRequest struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Type      string    `db:"type"`
	Payload   string    `db:"payload"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

type ModerationItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	CreatedBy  string `json:"createdBy"`
	AssignedTo string `json:"assignedTo"`
	Status     string `json:"status"`
}

type ModerationListResponse struct {
	Items      []ModerationItem `json:"items"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

type ModerationListQuery struct {
	Validator
	Page       *int   `json:"-"`
	PageSize   *int   `json:"-"`
	Status     string `json:"-"`
	Type       string `json:"-"`
	AssignedTo string `json:"-"`
}

func inStringSlice(v string, xs []string) bool {
	for _, x := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func (q ModerationListQuery) Validate() map[string]string {
	errs := map[string]string{}

	if q.Page != nil {
		if *q.Page < 1 || *q.Page > 1000 {
			errs["page"] = ErrInvalidField
		}
	}
	if q.PageSize != nil {
		if *q.PageSize < 10 || *q.PageSize > 500 {
			errs["pageSize"] = ErrInvalidField
		}
	}
	if q.Status != "" && !inStringSlice(q.Status, AllowedModerationStatuses) {
		errs["status"] = ErrInvalidField
	}
	if q.Type != "" && !inStringSlice(q.Type, AllowedModerationTypes) {
		errs["type"] = ErrInvalidField
	}
	return errs
}
