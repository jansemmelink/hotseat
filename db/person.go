package db

import (
	"fmt"
	"time"

	"github.com/go-msvc/errors"
	"github.com/google/uuid"
)

type Person struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Surname       string        `json:"surname"`
	Dob           *SqlDate      `json:"dob,omitempty"`
	Gender        *SqlGender    `json:"gender,omitempty"`
	Email         *string       `json:"email,omitempty"`
	Phone         *string       `json:"phone,omitempty"`
	Address       *Address      `json:"address,omitempty"`
	Nationalities []Nationality `json:"nationalities,omitempty"`
	Parents       []*Person     `json:"parent,omitempty"`
	Children      []*Person     `json:"children,omitempty"`
}

type Nationality struct {
	Country    *Country `json:"country"`
	NationalID string   `json:"national_id"`
}

type personRow struct {
	ID      string     `db:"id"`
	Name    string     `db:"name"`
	Surname string     `db:"surname"`
	Dob     *SqlDate   `db:"dob"`
	Gender  *SqlGender `db:"gender"`
	Email   *string    `db:"email,omitempty"`
	Phone   *string    `db:"phone,omitempty"`
}

func (pr personRow) Person() Person {
	p := Person{
		ID:            pr.ID,
		Name:          pr.Name,
		Surname:       pr.Surname,
		Dob:           pr.Dob,
		Gender:        pr.Gender,
		Email:         pr.Email,
		Phone:         pr.Phone,
		Address:       nil,
		Nationalities: nil,
		Parents:       nil,
		Children:      nil,
	}
	return p
}

const personRowQuery = "SELECT id,name,surname,dob,gender,email,phone FROM persons"

type PersonsFilter struct {
	Name    *string
	Surname *string
}

func GetPersons(filter PersonsFilter, sort []string, limit int) ([]Person, error) {
	log.Debugf("GetPersons(filter:%+v, sort:%+v, limit:%v)", filter, sort, limit)
	filterQuery := []string{}
	filterArgs := map[string]interface{}{}
	if filter.Name != nil && *filter.Name != "" {
		filterQuery = append(filterQuery, "name like %%:name%%")
		filterArgs["name"] = *filter.Name
	}
	if filter.Surname != nil && *filter.Surname != "" {
		filterQuery = append(filterQuery, "surname like %%:surname%%")
		filterArgs["surname"] = *filter.Surname
	}
	// 	case "email":
	// 		filterQuery = append(filterQuery, "email like %%:email%%")
	// 		filterArgs["email"] = v
	// 	case "phone":
	// 		filterQuery = append(filterQuery, "phone like %%:phone%%")
	// 		filterArgs["phone"] = v
	// 	case "gender":
	// 		filterQuery = append(filterQuery, "gender=:gender")
	// 		filterArgs["gender"] = v
	// 	case "dob_before":
	// 		filterQuery = append(filterQuery, "dob<=:dob_before")
	// 		filterArgs["dob_before"] = v
	// 	case "dob_after":
	// 		filterQuery = append(filterQuery, "dob>=:dob_after")
	// 		filterArgs["dob_after"] = v
	// 	default:
	// 		return nil, errors.Errorf("unknown filter field \"%s\"", n)
	// 	}
	// }

	query := personRowQuery
	for i, f := range filterQuery {
		if i == 0 {
			query += " where " + f
		} else {
			query += " and " + f
		}
	}
	if len(sort) > 0 {
		query += " order by "
		for i, s := range sort {
			if i > 0 {
				query += ","
			}
			query += s
		}
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	query += fmt.Sprintf(" limit %d", limit)

	var personRows []personRow
	if err := NamedSelect(&personRows, query, filterArgs); err != nil {
		return nil, errors.Wrapf(err, "failed to select persons")
	}

	persons := make([]Person, len(personRows))
	for i, pr := range personRows {
		persons[i] = pr.Person()
	}
	return persons, nil
}

func GetPerson(id string) (*Person, error) {
	var pr personRow
	if err := NamedGet(&pr, personRowQuery+" where id=:id", map[string]interface{}{"id": id}); err != nil {
		return nil, errors.Errorf("failed to read person record")
	}
	person := pr.Person()
	return &person, nil
}

//like upsert, returns existing person if found
func AddPersonIfNotExist(p Person) (*Person, error) {
	//todo: use common GetPerson with this key instead of ID
	var existingPersonRow personRow
	if err := NamedGet(
		&existingPersonRow,
		"SELECT id FROM persons WHERE name=:name AND surname=:surname AND dob=:dob AND gender=:gender", //check for duplicate on unique key
		map[string]interface{}{
			"name":    p.Name,
			"surname": p.Surname,
			"gender":  *p.Gender,
			"dob":     *p.Dob,
		},
	); err != nil {
		log.Debugf("did not get duplicate person, adding... %+v", err)
		return AddPerson(p)
	}
	p.ID = existingPersonRow.ID
	log.Debugf("found existing person: %+v", p)
	return &p, nil
}

func AddPerson(p Person) (*Person, error) {
	if p.Name == "" {
		return nil, errors.Errorf("missing name")
	}
	if p.Surname == "" {
		return nil, errors.Errorf("missing surname")
	}
	if p.Email == nil || !ValidEmail(*p.Email) {
		return nil, errors.Errorf("missing/invalid email")
	}
	if p.Phone == nil || *p.Phone == "" {
		return nil, errors.Errorf("missing phone")
	}
	if p.Dob == nil || time.Time(*p.Dob).Before(registerDobMin) || time.Time(*p.Dob).After(time.Now()) {
		return nil, errors.Errorf("missing or invalid dob")
	}
	if p.Gender == nil || (*p.Gender != SqlGenderMale && *p.Gender != SqlGenderFemale) {
		return nil, errors.Errorf("missing or invalid gender")
	}
	if len(p.Nationalities) < 1 {
		return nil, errors.Errorf("missing nationalities")
	}

	p.ID = uuid.New().String()
	if _, err := db.NamedExec(
		"INSERT INTO `persons` SET id=:id,name=:name,surname=:surname,gender=:gender,dob=:dob,email=:email,phone=:phone",
		map[string]interface{}{
			"id":      p.ID,
			"name":    p.Name,
			"surname": p.Surname,
			"gender":  *p.Gender,
			"dob":     *p.Dob,
			"email":   *p.Email,
			"phone":   *p.Phone,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to create person record")
	}

	for _, n := range p.Nationalities {
		if _, err := db.NamedExec(
			"INSERT INTO person_nationalities SET person_id=:pid,country_id=:cid,national_id=:nid",
			map[string]interface{}{
				"pid": p.ID,
				"cid": n.Country.ID,
				"nid": n.NationalID,
			},
		); err != nil {
			return nil, errors.Wrapf(err, "failed to create nationality record")
		}
	}

	return &p, nil
}
