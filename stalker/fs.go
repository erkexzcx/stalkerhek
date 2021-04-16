package stalker

import (
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config contains configuration taken from the YAML file.
type Config struct {
	Portal *Portal `yaml:"portal"`
	HLS    struct {
		Enabled bool   `yaml:"enabled"`
		Bind    string `yaml:"bind"`
	} `yaml:"hls"`
	Proxy struct {
		Enabled   bool   `yaml:"enabled"`
		Bind      string `yaml:"bind"`
		Rewrite   bool   `yaml:"rewrite"`
		RewriteTo string `yaml:"rewrite_to"`
	} `yaml:"proxy"`
}

// Portal represents Stalker portal
type Portal struct {
	Model        string `yaml:"model"`
	SerialNumber string `yaml:"serial_number"`
	DeviceID     string `yaml:"device_id"`
	DeviceID2    string `yaml:"device_id2"`
	Signature    string `yaml:"signature"`
	MAC          string `yaml:"mac"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Location     string `yaml:"url"`
	TimeZone     string `yaml:"time_zone"`
	Token        string `yaml:"token"`
}

// ReadConfig returns configuration from the file in Portal object
func ReadConfig(path *string) (*Config, error) {
	content, err := ioutil.ReadFile(*path)
	if err != nil {
		return nil, err
	}

	var c *Config
	err = yaml.Unmarshal(content, &c)
	if err != nil {
		return nil, err
	}

	if err = c.validateWithDefaults(); err != nil {
		return nil, err
	}
	return c, nil
}

var regexMAC = regexp.MustCompile(`^[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}:[A-F0-9]{2}$`)
var regexTimezone = regexp.MustCompile(`^[a-zA-Z]+/[a-zA-Z]+$`)

func (c *Config) validateWithDefaults() error {
	c.Portal.MAC = strings.ToUpper(c.Portal.MAC)

	if c.Portal.Model == "" {
		return errors.New("empty model")
	}

	if c.Portal.SerialNumber == "" {
		return errors.New("empty serial number (sn)")
	}

	if c.Portal.DeviceID == "" {
		return errors.New("empty device_id")
	}

	if c.Portal.DeviceID2 == "" {
		return errors.New("empty device_id2")
	}

	// Signature can be empty and it's fine

	if !regexMAC.MatchString(c.Portal.MAC) {
		return errors.New("invalid MAC '" + c.Portal.MAC + "'")
	}

	/* Username and password fields are optional */

	if c.Portal.Location == "" {
		return errors.New("empty portal url")
	}

	if !regexTimezone.MatchString(c.Portal.TimeZone) {
		return errors.New("invalid timezone '" + c.Portal.TimeZone + "'")
	}

	if !c.HLS.Enabled && !c.Proxy.Enabled {
		return errors.New("no services enabled")
	}

	if c.HLS.Enabled && c.HLS.Bind == "" {
		return errors.New("empty HLS bind")
	}

	if c.Proxy.Enabled && c.Proxy.Bind == "" {
		return errors.New("empty proxy bind")
	}

	if c.Proxy.Rewrite {
		if !c.HLS.Enabled {
			return errors.New("HLS service must be enabled for 'proxy: rewrite'")
		}

		_, _, err := net.SplitHostPort(c.Proxy.RewriteTo)
		if c.Proxy.RewriteTo != "" && err != nil {
			return errors.New("invalid 'proxy: rewrite_to' value")
		}
	}

	if c.Portal.Token == "" {
		c.Portal.Token = randomToken()
		log.Println("No token given, using random one:", c.Portal.Token)
	}

	return nil
}

func randomToken() string {
	allowlist := []rune("ABCDEF0123456789")
	b := make([]rune, 32)
	for i := range b {
		b[i] = allowlist[rand.Intn(len(allowlist))]
	}
	return string(b)
}
