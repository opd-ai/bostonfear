package main

import (
	"log"

	rootcmd "github.com/opd-ai/bostonfear/cmd"
)

func main() {
	if err := rootcmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
