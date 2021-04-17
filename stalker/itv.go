package stalker

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"
)

type ITV struct {
	Portal    *Portal // Reference to Stalker portal from where this ITV is taken
	CMDString string  // channel's identifier in Stalker portal

	// Additional information
	Title     string
	CMD_ID    string
	CMD_CH_ID string
}

// RetrieveListOfITVs retrieves all TV channels from stalker portal.
func (p *Portal) RetrieveListOfITVs() []*ITV {
	tmp := struct {
		Js struct {
			Data []struct {
				Name string `json:"name"` // Title of ITV
				Cmd  string `json:"cmd"`  // Some sort of URL used to request ITV real URL
				CMDs []struct {
					ID    string `json:"id"`    // Used for Proxy service to generate fake response to new URL request
					CH_ID string `json:"ch_id"` // Used for Proxy service to generate fake response to new URL request
				} `json:"cmds"`
			} `json:"data"`
		} `json:"js"`
	}{}

	content, err := p.request(p.Location+"?type=itv&action=get_all_channels&force_ch_link_check=&JsHttpRequest=1-xml", 3)
	if err != nil {
		log.Fatalln(err)
	}

	// Dump json output to file
	//ioutil.WriteFile("/tmp/dumpedchannels.json", content, 0644)

	if err := json.Unmarshal(content, &tmp); err != nil {
		log.Fatalln(string(content))
	}

	// Build ITVs list and return
	itvs := make([]*ITV, 0, len(tmp.Js.Data))
	for _, v := range tmp.Js.Data {
		itvs = append(itvs, &ITV{
			Portal:    p,
			CMDString: v.Cmd,

			Title:     v.Name,
			CMD_CH_ID: v.CMDs[0].ID,
			CMD_ID:    v.CMDs[0].CH_ID,
		})
	}

	return itvs
}

// GenerateLink retrieves a link from Stalker portal. It will expire very soon if not used.
func (i *ITV) GenerateLink() (string, error) {
	tmp := struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}{}

	link := i.Portal.Location + "?action=create_link&type=itv&cmd=" + url.PathEscape(i.CMDString) + "&JsHttpRequest=1-xml"
	content, err := i.Portal.request(link, 3)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(content, &tmp); err != nil {
		return "", err
	}

	strs := strings.Split(tmp.Js.Cmd, " ")
	return strs[len(strs)-1], nil
}

// CMD returns CMD string of ITV.
func (i *ITV) CMD() string {
	return i.CMDString
}

func (i *ITV) GenerateRewrittenResponse(destination string) string {
	return `{"js":{"id":"` + i.CMD_ID + `","cmd":"` + specialLinkEscape(destination) + `","streamer_id":0,"link_id":` + i.CMD_CH_ID + `,"load":0,"error":""},"text":"array(6) {\n  [\"id\"]=>\n  string(4) \"` + i.CMD_ID + `\"\n  [\"cmd\"]=>\n  string(99) \"` + specialLinkEscape(destination) + `\"\n  [\"streamer_id\"]=>\n  int(0)\n  [\"link_id\"]=>\n  int(` + i.CMD_CH_ID + `)\n  [\"load\"]=>\n  int(0)\n  [\"error\"]=>\n  string(0) \"\"\n}\ngenerated in: 0.01s; query counter: 8; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`
}
