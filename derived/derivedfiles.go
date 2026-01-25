package derived

import (
	"github.com/KitchenMishap/pudding-shed/chainreadinterface"
	"github.com/KitchenMishap/pudding-shed/chainstorage"
)

type DerivedFiles struct {
	folder          string
	privilegedFiles chainstorage.IPrivilegedFiles
	cri             chainreadinterface.IBlockChain
}

func (df *DerivedFiles) OpenReadOnly() error {
	return nil
}

func (df *DerivedFiles) PrivilegedFiles() chainstorage.IPrivilegedFiles { return df.privilegedFiles }
