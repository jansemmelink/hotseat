package db

type Session struct {
	Token       string   `json:"token"`
	User        User     `json:"user"`
	TimeCreated *SqlTime `json:"time_created,omitempty"`
	TimeUpdated *SqlTime `json:"time_updates,omitempty"`
}
