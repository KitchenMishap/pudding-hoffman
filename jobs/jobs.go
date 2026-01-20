package jobs

import (
	"github.com/KitchenMishap/pudding-huffman/blockchain"
)

func GatherStatistics(folder string) error {
	println("Please wait... opening files")
	reader, err := blockchain.NewChainReader(folder)
	if err != nil {
		return err
	}
	println("Finished opening files.")
	latestBlock, err := reader.Blockchain().LatestBlock()
	if err != nil {
		return err
	}
	println("The last block height is:", latestBlock.Height())
	return nil
}
