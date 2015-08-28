package main

import (
	"flag"
	"log"
	"os"

	"github.com/Patrolavia/ggpk/afs"
)

func main() {
	flag.Parse()
	fn := flag.Arg(0)
	f, err := os.Open(fn)
	if err != nil {
		log.Fatalf("Cannot open Content.ggpk at %s: %s", fn, err)
	}
	defer f.Close()

	root, err := afs.FromGGPK(f)
	if err != nil {
		log.Fatalf("Parse error: %s", err)
	}

	//root.Dump()
	_ = root
}