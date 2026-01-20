package jobs

import (
	"errors"
	"github.com/KitchenMishap/pudding-huffman/blockchain"
)

func GatherStatistics(folder string) error {
	println("Please wait... opening files")
	reader, err := blockchain.NewChainReader(folder)
	if err != nil {
		return err
	}
	println("Finished opening files.")
	blockchainInterface := reader.Blockchain()
	latestBlock, err := blockchainInterface.LatestBlock()
	if err != nil {
		return err
	}
	println("The last block height is:", latestBlock.Height())

	blocks := latestBlock.Height() + 1
	blockToTxo := make([]int64, blocks)

	println("Gathering txo indices for each block...")
	blockHeight := int64(0)
	blockHandle := blockchainInterface.GenesisBlock()
	for {
		if blockHeight%100000 == 0 {
			println("Block: ", blockHeight)
		}

		block, err := blockchainInterface.BlockInterface(blockHandle)
		if err != nil {
			return err
		}
		transHandle, err := block.NthTransaction(0)
		if err != nil {
			return err
		}
		trans, err := blockchainInterface.TransInterface(transHandle)
		if err != nil {
			return err
		}
		txoHandle, err := trans.NthTxo(0)
		if err != nil {
			return err
		}
		if !txoHandle.TxoHeightSpecified() {
			return errors.New("txo height not specified by handle")
		}
		blockToTxo[blockHeight] = txoHandle.TxoHeight()

		blockHeight++
		if blockHeight == blocks {
			break
		}
		blockHandle, err = blockchainInterface.NextBlock(blockHandle)
	}
	println("There are: ", blockToTxo[blocks-1], " txos in the first: ", blocks, " blocks.")

	return nil
}
