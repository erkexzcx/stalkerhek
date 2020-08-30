package config

import (
	"errors"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config stores configuration from yaml file
type Config struct {
	Model        string `yaml:"model"`
	SerialNumber string `yaml:"serial_number"`
	DeviceID     string `yaml:"device_id"`
	DeviceID2    string `yaml:"device_id2"`
	Signature    string `yaml:"signature"`
	MAC          string `yaml:"mac"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Location     string `yaml:"portal_url"`
	TimeZone     string `yaml:"time_zone"`
	Token        string `yaml:"token"`
}

// Load returns configuration object.
func Load(path *string) (*Config, error) {
	content, err := ioutil.ReadFile(*path)
	if err != nil {
		return nil, err
	}

	var c *Config
	err = yaml.Unmarshal(content, &c)
	if err != nil {
		return nil, err
	}

	if err = c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate checks for errors in config file
func (c *Config) Validate() error {
	if c.Model != "MAG250" && c.Model != "MAG254" {
		return errors.New("only supported models are MAG250 and MAG254")
	}
	if strings.Replace(c.MAC, " ", "", 1) != c.MAC {
		return errors.New("MAC cannot contain spaces")
	}
	if c.MAC == "" {
		return errors.New("MAC cannot be empty")
	}
	if !strings.HasSuffix(c.Location, ".php") {
		return errors.New("invalid Stalker portal location: it must end with '.php'")
	}

	if strings.Replace(c.TimeZone, " ", "", 1) != c.TimeZone {
		return errors.New("timezone cannot contain spaces")
	}
	if c.TimeZone == "" {
		return errors.New("timezone cannot be empty")
	}

	if strings.Replace(c.Token, " ", "", 1) != c.Token {
		return errors.New("token cannot contain spaces")
	}
	if c.Token == "" {
		return errors.New("token cannot be empty")
	}
	return nil
}
