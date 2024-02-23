package file

import (
	"encoding/hex"
	"img-lab/pkg/util"
)

type HashInfo struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"short_hash"`
}

func (i *Info) GetHashInfo(log func(string, ...any)) error {
	log("Hashing " + i.Path)

	hash, err := util.Md5HashOfFile(i.Path)
	if err != nil {
		return err
	}

	hashHex := hex.EncodeToString(hash)
	shortHashHex := hashHex[:8]

	i.HashInfo = &HashInfo{
		Hash:      hashHex,
		ShortHash: shortHashHex,
	}

	return nil
}
