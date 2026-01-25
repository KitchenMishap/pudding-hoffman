package blockchain

// This file is related but different to the blockchain/chain.go in pudding-grid on which it is based

import (
	"github.com/KitchenMishap/pudding-shed/chainreadinterface"
	"github.com/KitchenMishap/pudding-shed/chainstorage"
)

type ChainReader struct {
	folder        string
	chainRead     chainreadinterface.IBlockChain
	handleCreator chainreadinterface.IHandleCreator
	parents       chainstorage.IParents
	privileged    chainstorage.IPrivilegedFiles
}

func NewChainReader(folder string) (*ChainReader, error) {
	reader := ChainReader{}
	reader.folder = folder
	creator, err := chainstorage.NewConcreteAppendableChainCreator(
		folder,
		[]string{"time", "mediantime", "difficulty", "strippedsize", "size", "weight"},
		[]string{"size", "vsize", "weight"},
		true, true)
	if err != nil {
		return nil, err
	}
	readableChain, handleCreator, parents, privileged, _ := creator.OpenReadOnly()
	reader.chainRead = readableChain
	reader.handleCreator = handleCreator
	reader.parents = parents
	reader.privileged = privileged
	return &reader, nil
}

func (cr ChainReader) Blockchain() chainreadinterface.IBlockChain { return cr.chainRead }
func (cr ChainReader) HandleCreator() chainreadinterface.IHandleCreator {
	return cr.handleCreator
}
func (cr ChainReader) Parents() chainstorage.IParents {
	return cr.parents
}
func (cr ChainReader) Privileged() chainstorage.IPrivilegedFiles { return cr.privileged }
