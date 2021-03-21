package stalker

import (
	"errors"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

// ReadConfig returns configuration from the file in Portal object
func ReadConfig(path *string) (*Portal, error) {
	content, err := ioutil.ReadFile(*path)
	if err != nil {
		return nil, err
	}

	var p *Portal
	err = yaml.Unmarshal(content, &p)
	if err != nil {
		return nil, err
	}

	if err = p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate validates checks
func (p *Portal) Validate() error {
	// ### I have no clue about the differences between these 2...
	// if p.Model != "MAG250" && p.Model != "MAG254" {
	// 	return errors.New("only supported models are MAG250 and MAG254")
	// }
	if strings.Replace(p.MAC, " ", "", 1) != p.MAC {
		return errors.New("MAC cannot contain spaces")
	}
	if p.MAC == "" {
		return errors.New("MAC cannot be empty")
	}
	if !strings.HasSuffix(p.Location, ".php") {
		return errors.New("invalid Stalker portal location: it must end with '.php'")
	}

	if strings.Replace(p.TimeZone, " ", "", 1) != p.TimeZone {
		return errors.New("timezone cannot contain spaces")
	}
	if p.TimeZone == "" {
		return errors.New("timezone cannot be empty")
	}

	if strings.Replace(p.Token, " ", "", 1) != p.Token {
		return errors.New("token cannot contain spaces")
	}
	if p.Token == "" {
		return errors.New("token cannot be empty")
	}
	return nil
}
