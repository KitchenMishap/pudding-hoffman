package blockchain

// This file is related but different to the blockchain/interfaces.go in pudding-grid on which it is based

import (
	"github.com/KitchenMishap/pudding-shed/chainreadinterface"
	"github.com/KitchenMishap/pudding-shed/chainstorage"
)

type AccessChain interface {
	Blockchain() chainreadinterface.IBlockChain
	HandleCreator() chainreadinterface.IHandleCreator
	Parents() chainstorage.IParents
}
