package derived

import (
	"github.com/KitchenMishap/pudding-shed/chainstorage"
)

type DerivedFiles struct {
	folder          string
	privilegedFiles chainstorage.IPrivilegedFiles
}

func NewDerivedFiles(folder string) (*DerivedFiles, error) {
	result := DerivedFiles{}
	result.folder = folder

	readCreator, err := chainstorage.NewConcreteAppendableChainCreator(folder, []string{}, []string{}, false)
	if err != nil {
		return nil, err
	}
	_, _, _, files, err := readCreator.OpenReadOnly()
	if err != nil {
		return nil, err
	}
	result.privilegedFiles = files

	return &result, nil
}

func (df *DerivedFiles) OpenReadOnly() error {
	return nil
}

func (df *DerivedFiles) PrivilegedFiles() chainstorage.IPrivilegedFiles { return df.privilegedFiles }
