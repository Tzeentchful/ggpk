package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tzeentchful/ggpk/afs"
	"gopkg.in/kothar/brotli-go.v0/dec"
)

var ddsMagic = [3]byte{0x44, 0x44, 0x53}
var compressedMagic = [3]byte{0x43, 0x4D, 0x50}

var ddsLinks map[string][]string

var (
	recursive bool
	destDir   string
)

func init() {
	ddsLinks = make(map[string][]string)
	flag.BoolVar(&recursive, "r", false, "Recursive extract directory, ignored if extracting file.")
	flag.StringVar(&destDir, "d", ".", "Extract files to directory `N`.")
	flag.Parse()
}

func main() {
	fn := flag.Arg(0)
	path := flag.Arg(1)
	if path == "" {
		log.Fatalf("You have to specify path to extract.")
	}
	if path[len(path)-1] == "/"[0] {
		path = path[:len(path)-1]
	}
	f, err := os.Open(fn)
	if err != nil {
		log.Fatalf("Cannot open ggpk file at %s: %s", fn, err)
	}
	defer f.Close()

	log.Print("Parsing ggpk file ...")
	root, err := afs.FromGGPK(f)
	if err != nil {
		log.Fatalf("Parse error: %s", err)
	}

	if path == "" {
		saveDir(root, f)
		return
	}

	cur := root
	nodes := strings.Split(path[1:], "/")
Orz:
	for idx, node := range nodes {
		if idx == len(nodes)-1 {
			for _, file := range cur.Files {
				if file.Name == node {
					saveFile(file, f)
					return
				}
			}
		}

		for _, dir := range cur.Subfolders {
			if dir.Name == node {
				if idx == len(nodes)-1 {
					saveDir(dir, f)
					return
				}
				cur = dir
				continue Orz
			}
		}
		log.Fatalf("Cannot find %s in %s", node, cur.Path)

	}
}

func saveFile(file *afs.File, f *os.File) {
	fmt.Printf("Writing file %s ... ", file.Path)
	data, err := file.Content()
	if err != nil {
		log.Fatalf("While reading file %s: %s", file.Path, err)
	}
	
	if len(data) >= 3 && filepath.Ext(file.Path) == ".dds" {
		if !bytes.Equal(data[:3], ddsMagic[:]) {
			if data[0] == 0x2A {
				path := "/" + string(data[1:])
				osPath := filepath.FromSlash(destDir + path)
				if _, err := os.Stat(osPath); err == nil {
					data, _ = ioutil.ReadFile(osPath)
					fmt.Printf("\nFileLink Exists: %s \n", osPath)
				} else {
					ddsLinks[path] = append(ddsLinks[path], file.Path)
					fmt.Printf("\nAdding Path: %s \n", path)
					return
				}
			} else {
				compressLen := binary.LittleEndian.Uint32(data[:4])
				data = decompressFile(data[4:], compressLen)
			}
		}
	} else if len(data) >= 3 && bytes.Equal(data[:3], compressedMagic[:]) {
		compressLen := binary.LittleEndian.Uint32(data[3:7])
		data = decompressFile(data[7:], compressLen)
	}

	towrite := []string{}
	towrite = append(towrite, file.Path)
	if len(ddsLinks[file.Path]) > 0 {
		fmt.Printf("we have a links \n")
		towrite = append(towrite, ddsLinks[file.Path]...)
	}

	for i := 0; i < len(towrite); i++ {
		fn := filepath.FromSlash(destDir + towrite[i])
		dirname := filepath.Dir(fn)
		if err := os.MkdirAll(dirname, os.FileMode(0777)); err != nil {
			log.Fatalf("Cannot create directory %s: %s", dirname, err)
		}

		dest, err := os.Create(fn)
		if err != nil {
			log.Fatalf("Error creating file %s: %s", file.Path, err)
		}
		defer dest.Close()

		if _, err := dest.Write(data); err != nil {
			log.Fatalf("Error writing file %s: %s", file.Path, err)
		}
		fmt.Printf("%d bytes\n", file.Size)
	}
}

func saveDir(dir *afs.Directory, f *os.File) {
	for _, file := range dir.Files {
		saveFile(file, f)
	}

	if recursive {
		for _, child := range dir.Subfolders {
			saveDir(child, f)
		}
	}
}

func decompressFile(compressed []byte, expectedLen uint32) []byte {
	decompressed, _ := dec.DecompressBuffer(compressed, make([]byte, expectedLen))

	/*if uint32(len(decompressed)) != expectedLen {
		log.Fatalf("Error decompressing DDS  expected size: %d decompressed size: %d", expectedLen, len(decompressed))
	}*/

	return decompressed
}
