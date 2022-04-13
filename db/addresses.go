package db

type Address struct {
	Phone  string  `json:"phone"`
	Street string  `json:"street"`
	Info   string  `json:"info,omitempty"`
	City   string  `json:"city"`
	Region *Region `json:"region"`
	Code   string  `json:"code,omitempty"`
}
