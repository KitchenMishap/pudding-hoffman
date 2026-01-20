package blockchain

// This file is related but different to the blockchain/chain.go in pudding-grid on which it is based

import (
	"github.com/KitchenMishap/pudding-huffman/derived"
	"github.com/KitchenMishap/pudding-shed/chainreadinterface"
	"github.com/KitchenMishap/pudding-shed/chainstorage"
)

type ChainReader struct {
	folder        string
	chainRead     chainreadinterface.IBlockChain
	handleCreator chainreadinterface.IHandleCreator
	parents       chainstorage.IParents
	derived       *derived.DerivedFiles
}

func NewChainReader(folder string) (*ChainReader, error) {
	reader := ChainReader{}
	reader.folder = folder
	creator, err := chainstorage.NewConcreteAppendableChainCreator(
		folder,
		[]string{"time", "mediantime", "difficulty", "strippedsize", "size", "weight"},
		[]string{"size", "vsize", "weight"},
		true)
	if err != nil {
		return nil, err
	}
	readableChain, handleCreator, parents, _, _ := creator.OpenReadOnly()
	reader.chainRead = readableChain
	reader.handleCreator = handleCreator
	reader.parents = parents
	derivedFiles, err := derived.NewDerivedFiles(folder)
	if err != nil {
		return nil, err
	}
	reader.derived = derivedFiles
	err = reader.derived.OpenReadOnly()
	if err != nil {
		return nil, err
	}
	return &reader, nil
}

func (cr ChainReader) Blockchain() chainreadinterface.IBlockChain { return cr.chainRead }
func (cr ChainReader) HandleCreator() chainreadinterface.IHandleCreator {
	return cr.handleCreator
}
func (cr ChainReader) Parents() chainstorage.IParents {
	return cr.parents
}
