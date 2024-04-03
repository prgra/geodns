package consulcfg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	Folder  string
	Client  *http.Client
}

var GClient *Client

func NewClient(address, folder string) *Client {
	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")
	if folder != "" {
		folder = fmt.Sprintf("%s/", folder)
	}
	GClient = &Client{
		Address: address,
		Folder:  folder,
		Client:  &http.Client{},
	}
	return GClient
}

func (c *Client) ReadConfig() {
	cfgurl := fmt.Sprintf("%s/v1/kv/%sgeodns.conf", c.Address, c.Folder)
	log.Println("get config from %s", cfgurl)
	res, err := http.Get(cfgurl)
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

	err = appconfig.ConfigReaderFromReader(cnf)
	if err != nil {
		log.Println("error reading consul config: ", err)
	}
}

// GetFileContent get binary content from file
func GetFileContent(fn string) (b []byte, err error) {
	log.Printf("use backup for %s", fn)
	f, err := os.Open(filepath.Clean(fn))
	if err != nil {
		return nil, err
	}
	b, err = io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return b, nil
}

func (c *Client) GetZoneData(fileName string) ([]byte, error) {
	if c.Address == "" {
		return nil, fmt.Errorf("consul address not set")
	}
	rawfilename := filepath.Base(fileName)
	zn := strings.TrimPrefix(rawfilename, "consul_")
	zn = strings.TrimSuffix(zn, ".json")
	zurl := fmt.Sprintf("%s/v1/kv/%s%s.json", c.Address, c.Folder, zn)
	log.Printf("get zone from consul %s", zurl)
	res, err := http.Get(zurl)
	if err != nil {
		log.Printf("can't connect to consul %s", err.Error())
		bak, ferr := GetFileContent(fileName)
		if ferr != nil || len(bak) == 0 {
			log.Printf("error read backup %s: %s", fileName, ferr)
			return nil, err
		}
		return bak, nil
	}
	if res.StatusCode != http.StatusOK {
		bak, ferr := GetFileContent(fileName)
		if ferr != nil || len(bak) == 0 {
			log.Printf("error read backup %s: %s", fileName, ferr)
			return nil, err
		}
		return bak, nil
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		bak, ferr := GetFileContent(fileName)
		if ferr != nil || len(bak) == 0 {
			log.Printf("error read backup %s: %s", fileName, ferr)
			return nil, err
		}
		return bak, nil
	}
	_ = res.Body.Close()
	var cresp []ConsulKVResponse
	err = json.Unmarshal(b, &cresp)
	if err != nil {
		return nil, err
	}
	if len(cresp) == 0 {
		return nil, fmt.Errorf("no data")
	}
	bi, err := base64.StdEncoding.DecodeString(cresp[0].Value)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(fileName, bi, 0o600)
	if err != nil {
		log.Printf("can't write file %s :%s", fileName, err.Error())
	}
	return bi, err
}
