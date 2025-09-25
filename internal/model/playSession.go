package model

import (
	"time"
)

var VisibilityPublic = "Public"
var VisibilityPrivate = "Private"

var DnD5eRS = "dnd5e"
var PathfinderRS = "pathfinder"
var Pathfinder2eRS = "pathfinder2e"
var CustomRS = "custom"

var Active = "Active"
var Archived = "Archived"

type PlaySessionDB struct {
	ID          string     `db:"id"`
	OwnerID     string     `db:"owner_id"`
	Title       string     `db:"title"`
	Ruleset     string     `db:"ruleset"`
	Description string     `db:"description"`
	Capacity    int        `db:"capacity"`
	Visibility  string     `db:"visibility"`
	StartsAt    *time.Time `db:"starts_at"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
}

type PlaySessionMemberDB struct {
	SessionID string `db:"session_id"`
	UserID    string `db:"user_id"`
	Role      string `db:"role"`
	Muted     bool   `db:"muted"`
}

type CreatePlaySessionDTO struct {
	Validator

	Title       string    `json:"title"`
	Ruleset     string    `json:"ruleset"`
	Description string    `json:"description"`
	Capacity    int       `json:"capacity"`
	Visibility  string    `json:"visibility"`
	StartsAt    time.Time `json:"startsAt"`
}

var AllowedRuleSets = []string{DnD5eRS, PathfinderRS, Pathfinder2eRS, CustomRS}
var AllowedVisibility = []string{VisibilityPublic, VisibilityPrivate}

func (dto CreatePlaySessionDTO) Validate() map[string]string {
	errs := map[string]string{}

	if dto.Title == "" {
		errs["title"] = ErrEmptyField
	}
	if !inStringSlice(dto.Ruleset, AllowedRuleSets) {
		errs["ruleset"] = ErrInvalidField
	}
	if dto.Capacity < 2 || dto.Capacity > 10 {
		errs["capacity"] = ErrInvalidField
	}
	if !inStringSlice(dto.Visibility, AllowedVisibility) {
		errs["visibility"] = ErrInvalidField
	}

	return errs
}
