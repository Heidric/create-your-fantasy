package model

import "time"

type MessageRow struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Text      string    `db:"text"`
	CreatedAt time.Time `db:"created_at"`
	SeqID     int       `db:"seq_id"`
}

type MessagesQuery struct {
	Validator
	LastMessage *int `json:"-"`
	PageSize    *int `json:"-"`
}

type SendMessageDTO struct {
	Validator
	Text string `json:"text"`
}

func (q MessagesQuery) Validate() map[string]string {
	errs := map[string]string{}
	if q.LastMessage != nil && *q.LastMessage < 0 {
		errs["lastMessage"] = ErrInvalidField
	}
	if q.PageSize != nil && (*q.PageSize < 10 || *q.PageSize > 200) {
		errs["pageSize"] = ErrInvalidField
	}
	return errs
}

func (d SendMessageDTO) Validate() map[string]string {
	errs := map[string]string{}
	if d.Text == "" {
		errs["text"] = ErrEmptyField
	}
	if len([]rune(d.Text)) > 2000 {
		errs["text"] = ErrInvalidField
	}
	return errs
}

type MessageItem struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessagesResponse struct {
	Items       []MessageItem `json:"items"`
	LastMessage int           `json:"lastMessage"`
}
