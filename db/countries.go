package db

import "github.com/go-msvc/errors"

type Country struct {
	ID                     string  `json:"id" db:"id"`
	Name                   string  `json:"name" db:"name"`
	NationalIDRegexPattern *string `json:"national_id_regex_pattern,omitempty" db:"national_id_regex_pattern"`
}

func GetCountries(nameFilter string) ([]Country, error) {
	var list []Country
	if err := NamedSelect(
		&list,
		"select id,name from countries where name like :name_filter order by name",
		map[string]interface{}{
			"name_filter": "%" + nameFilter + "%",
		}); err != nil {
		return []Country{}, errors.Wrapf(err, "failed to get list of countries")
	}
	return list, nil
}

func GetCountryByID(id string) (*Country, error) {
	country := Country{}
	if err := NamedGet(
		&country,
		"select id,name,national_id_regex_pattern from countries where id=:id",
		map[string]interface{}{
			"id": id,
		}); err != nil {
		return nil, errors.Wrapf(err, "failed to get country by id(%s)", id)
	}
	return &country, nil
}

func GetCountryByName(name string) (*Country, error) {
	country := Country{}
	if err := NamedGet(
		&country,
		"select id,name,national_id_regex_pattern from countries where name=:name",
		map[string]interface{}{
			"name": name,
		}); err != nil {
		return nil, errors.Wrapf(err, "failed to get country by name(%s)", name)
	}
	return &country, nil
}

type Region struct {
	Country *Country `json:"country"`
	Name    string   `json:"name"`
	Code    string   `json:"code"`
}

func CountryRegions(countryID string, nameFilter string) ([]Region, error) {
	var list []Region
	if err := NamedSelect(&list, "select id,name from regions where country_id=? and name like ? order by name", nameFilter); err != nil {
		return []Region{}, errors.Wrapf(err, "failed to get list of regions")
	}
	return list, nil
}
