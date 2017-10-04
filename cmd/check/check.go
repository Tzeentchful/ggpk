package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Tzeentchful/ggpk/afs"
	"github.com/Tzeentchful/ggpk/record"
)

func fatal(i interface{}) {
	log.Fatal("[ERROR] ", i)
}

func fatalf(s string, args... interface{}) {
	log.Fatalf("[ERROR] " + s, args...)
}

func main() {
	flag.Parse()
	fn := flag.Arg(0)
	f, err := os.Open(fn)
	if err != nil {
		fatalf("Cannot open ggpk file %s: %s", fn, err)
	}
	defer f.Close()

	rootNode, err := record.GGG(f)
	if err != nil {
		fatalf("Cannot read ggpk signature: %s", err)
	}

	if rootNode.Header.Tag != "GGPK" {
		fatal("This file is not ggpk file, or corrupted.")
	}

	nodes, err := rootNode.Children(f)
	if err != nil {
		fatalf("Cannot read root node from ggpk: %s", err)
	}

	log.Print("Checking ...")
	for _, node := range nodes {
		doHeader(node, f, "")
	}
	log.Print("All ok.")
}

func doHeader(h record.RecordHeader, f *os.File, path string) (ret []byte) {
	switch h.Tag {
	case "PDIR":
		ret = doDir(h, f, path)
	case "FILE":
		ret = doFile(h, f, path)
	case "FREE":
		fmt.Println("Skip free space.")
	default:
		fatalf("Unknown record type %s", h.Tag)
	}
	return ret
}

func b(path string, digest []byte) {
	fmt.Printf("Checking %s (%x) ... ", path, digest)
}

func c(digest []byte, data []byte) {
	sum := sha256.Sum256(data)
	for k, v := range digest {
		if k >= len(sum) || v != sum[k] {
			fmt.Printf("%x\n", sum)
			fatal("Checksum mismatch!")
		}
	}
	fmt.Println("ok.")
}

func doFile(h record.RecordHeader, f *os.File, path string) []byte {
	r, err := record.ReadFile(f, h)
	if err != nil {
		fatalf("Cannot read file in %s: %s", path, err)
	}

	fn := path + r.Name
	b(fn, r.Digest)
	af := afs.FromFileRecord(h, r, 0)
	data, err := af.Content()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	c(r.Digest, data)
	return r.Digest
}

func doDir(h record.RecordHeader, f *os.File, path string) []byte {
	r, err := record.ReadDir(f, h)
	if err != nil {
		if path == "" {
			path = "ROOT"
		}
		fatalf("Cannot read directory in %s: %s", path, err)
	}

	fn := path + r.Name + "/"
	data := make([]byte, 0)
	for _, e := range r.Entries {
		data = append(data, doEntry(e, f, fn)...)
	}
	b(fn, r.Digest)
	c(r.Digest, data)
	return r.Digest
}

func doEntry(e record.DirectoryEntry, f *os.File, path string) []byte {
	if _, err := f.Seek(int64(e.Offset), 0); err != nil {
		fatalf("Cannot seek to %d: %s", e.Offset, err)
	}
	h, err := record.Header(f)
	if err != nil {
		fatalf("Cannot read header from %d: %s", e.Offset, err)
	}
	h.Offset = e.Offset + uint64(h.ByteLength())
	return doHeader(h, f, path)
}
