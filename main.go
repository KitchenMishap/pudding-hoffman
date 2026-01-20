package main

import (
	"flag"
	"github.com/KitchenMishap/pudding-huffman/jobs"
)

func main() {
	var sDirFlag = flag.String("Dir", "", "Directory to serve data from")
	flag.Parse()

	var err error
	err = jobs.GatherStatistics(*sDirFlag)

	if err != nil {
		println(err.Error())
	}
}
