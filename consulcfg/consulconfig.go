package consulcfg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/abh/geodns/v3/appconfig"
)

type ConsulKVResponse struct {
	LockIndex   int
	Key         string
	Flags       int
	Value       string
	CreateIndex int
	ModifyIndex int
}

type Client struct {
	Address string
	Client  *http.Client
}

var GClient *Client

func NewClient(address string) *Client {
	GClient = &Client{
		Address: address,
		Client:  &http.Client{},
	}
	return GClient
}

func (c *Client) ReadConfig() {
	res, err := http.Get(c.Address + "/v1/kv/geodns.conf")
	if err != nil {
		log.Println("error reading consul config: ", err)
	}
	if res.StatusCode != http.StatusOK {
		log.Println("error reading consul config: ", res.Status)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading consul config: ", err)
	}
	_ = res.Body.Close()
	var cresp []ConsulKVResponse
	err = json.Unmarshal(b, &cresp)
	if err != nil {
		log.Println("error reading consul config: ", err)
	}
	if len(cresp) == 0 {
		log.Println("error reading consul config: no data")
		return
	}
	cnf := base64.NewDecoder(base64.StdEncoding, strings.NewReader(cresp[0].Value))

	fmt.Println("Config:", cresp[0].Value)
	err = appconfig.ConfigReaderFromReader(cnf)
	if err != nil {
		log.Println("error reading consul config: ", err)
	}
}

func (c *Client) GetZoneData(zone string) ([]byte, error) {
	if c.Address == "" {
		return nil, fmt.Errorf("consul address not set")
	}
	res, err := http.Get(c.Address + "/v1/kv/" + zone)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, err
	}
	b, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	return b, err
}
