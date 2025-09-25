package model

type CreateReportDTO struct {
	Validator

	Type      string `json:"type"`
	SubjectID string `json:"subjectId"`
	Comment   string `json:"comment"`
}

var AllowedReportTypes = []string{
	"profileReport",
	"characterReport",
	"campaignReport",
	"messageReport",
}

func (dto CreateReportDTO) Validate() map[string]string {
	errs := map[string]string{}

	if dto.Type == "" {
		errs["type"] = ErrEmptyField
	} else {
		ok := false
		for _, t := range AllowedReportTypes {
			if dto.Type == t {
				ok = true
				break
			}
		}
		if !ok {
			errs["type"] = ErrInvalidField
		}
	}

	if dto.SubjectID == "" {
		errs["subjectId"] = ErrEmptyField
	}
	if dto.Comment == "" {
		errs["comment"] = ErrEmptyField
	}

	return errs
}
