package model

type ModerationReviewDTO struct {
	Validator
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

func (d ModerationReviewDTO) Validate() map[string]string {
	errs := map[string]string{}
	switch d.Decision {
	case "changesRequested", "subjectApproved", "subjectHidden":
	default:
		errs["decision"] = ErrInvalidField
	}
	return errs
}
