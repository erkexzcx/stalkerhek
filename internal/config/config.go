package config

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
	"gopkg.in/yaml.v2"
)

// Config stores configuration from yaml file
type Config struct {
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

// StalkerPortal provides configuration as Stalker portal
func (c *Config) StalkerPortal() *stalker.Portal {
	return &stalker.Portal{
		SerialNumber: c.SerialNumber,
		DeviceID:     c.DeviceID,
		DeviceID2:    c.DeviceID2,
		Signature:    c.Signature,
		MAC:          c.MAC,
		Username:     c.Username,
		Password:     c.Password,
		Location:     c.Location,
		TimeZone:     c.TimeZone,
		Token:        c.Token,
	}
}

// LoadConfig provides pointer to configuration
func LoadConfig() (*Config, error) {
	content, err := ioutil.ReadFile("stalkerhek.yaml")
	if err != nil {
		return nil, err
	}

	var c Config

	err = yaml.Unmarshal(content, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Validate checks for errors in config file
func (c *Config) Validate() error {
	/* Some very basic checks... */
	if strings.Replace(c.SerialNumber, " ", "", 1) != c.SerialNumber {
		return errors.New("serial number cannot contain spaces")
	}
	if c.SerialNumber == "" {
		return errors.New("serial number cannot be empty")
	}

	if strings.Replace(c.DeviceID, " ", "", 1) != c.DeviceID {
		return errors.New("device ID cannot contain spaces")
	}
	if c.DeviceID == "" {
		return errors.New("device ID cannot be empty")
	}

	if strings.Replace(c.DeviceID2, " ", "", 1) != c.DeviceID2 {
		return errors.New("device ID2 cannot contain spaces")
	}
	if c.DeviceID2 == "" {
		return errors.New("device ID2 cannot be empty")
	}

	if strings.Replace(c.Signature, " ", "", 1) != c.Signature {
		return errors.New("signature cannot contain spaces")
	}
	if c.Signature == "" {
		return errors.New("signature cannot be empty")
	}

	if strings.Replace(c.MAC, " ", "", 1) != c.MAC {
		return errors.New("MAC cannot contain spaces")
	}
	if c.MAC == "" {
		return errors.New("MAC cannot be empty")
	}

	if strings.Replace(c.Username, " ", "", 1) != c.Username {
		return errors.New("username cannot contain spaces")
	}
	if c.Username == "" {
		return errors.New("username cannot be empty")
	}

	if c.Password == "" {
		return errors.New("password cannot be empty")
	}

	if !strings.HasSuffix(c.Location, "/stalker_portal/") {
		return errors.New("invalid Stalker portal location: it must end with '/stalker_portal/'")
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
