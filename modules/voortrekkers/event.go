package voortrekkers

import "time"

type Event struct {
	StartTime time.Time
	EndTime   time.Time
	Address   Address
	Name      string
	Organizer *Person
	Contacts  map[string]*Person
	Cost      Amount
	Info      Document
	Qualify   []Rule
	Fields    []Field   //values needed to fill in, could be options for transport, or color preference selection, ...
	AddOns    []Product //products one can add to entry
	SubEvents []Event
}

type Amount float64

type Person struct{}

type Document struct{}

type Product struct{}

type Rule struct{}

type Field struct{}

type Address struct{}
