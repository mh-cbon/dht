package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mh-cbon/dht/util"
)

// DataSave is a type to store known ip and contacts of an existing dht session.
type DataSave struct {
	OldIP *util.CompactPeer
	Nodes []string
}

// Get loads session artifacts from a file.
func Get(file string) (*DataSave, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		return nil, fmt.Errorf("file does not exists %q in cwd %q", file, wd)
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ret := &DataSave{}
	decoder := json.NewDecoder(f)
	if decoder.Decode(ret) == nil {
		return ret, nil
	}
	return ret, decoder.Decode(&ret.Nodes)
}

// Save a bootstrap artifacts to a file.
func Save(file string, currentIP *util.CompactPeer, nodes []string) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "   ")
	return enc.Encode(DataSave{OldIP: currentIP, Nodes: nodes})
}
