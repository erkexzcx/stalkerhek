package stalker

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"strings"
)

// Channel stores information about channel in Stalker portal. This is not
// a real TV channel, but details on how to retrieve a working channel's URL.
type Channel struct {
	cmd    string  // channel's identifier in Stalker portal
	logo   string  // Full URL to logo in Stalker portal
	portal *Portal // Reference to portal from where this channel is taken from
}

// NewLink retrieves a link to the working channel. Retrieved link can
// be played in VLC or Kodi, but expires very soon if not being constantly
// opened (used).
func (c *Channel) NewLink() (string, error) {
	type tmpStruct struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}
	var tmp tmpStruct

	link := c.portal.Location + "server/load.php?action=create_link&type=itv&cmd=" + url.PathEscape(c.cmd) + "&JsHttpRequest=1-xml"
	content, err := c.portal.httpRequest(link)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(content, &tmp); err != nil {
		panic(err)
	}

	strs := strings.Split(tmp.Js.Cmd, " ")
	if len(strs) == 2 {
		return strs[1], nil
	}
	return "", errors.New("Stalker portal returned invalid link to TV Channel: " + tmp.Js.Cmd)
}

// Logo returns full link to channel's logo
func (c *Channel) Logo() string {
	if c.logo == "" {
		return ""
	}
	return c.portal.Location + "misc/logos/320/" + c.logo
}

// RetrieveChannels retrieves all TV channels from stalker portal.
func (p *Portal) RetrieveChannels() (map[string]*Channel, error) {
	type tmpStruct struct {
		Js struct {
			Data []struct {
				Name string `json:"name"`
				Cmd  string `json:"cmd"`
				Logo string `json:"logo"`
			} `json:"data"`
		} `json:"js"`
	}
	var tmp tmpStruct

	content, err := p.httpRequest(p.Location + "server/load.php?type=itv&action=get_all_channels&force_ch_link_check=&JsHttpRequest=1-xml")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(content, &tmp); err != nil {
		log.Println(string(content))
		panic(err)
	}

	channels := make(map[string]*Channel, len(tmp.Js.Data))
	for _, v := range tmp.Js.Data {
		channels[v.Name] = &Channel{
			cmd:    v.Cmd,
			logo:   v.Logo,
			portal: p,
		}
	}

	return channels, nil
}
