package main

import (
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	var err error
	var data []byte

	var commit []byte
	var tree []byte

	// Read master branch
	r, err := os.Open(".git/refs/heads/master")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	hs := strings.TrimSpace(string(buf))
	commit, err = hex.DecodeString(hs)
	if err != nil {
		panic(err)
	}

	println(string(commit))

	r, err = os.Open(fmt.Sprintf(".git/objects/%s/%s", hs[:2], hs[2:]))
	if err != nil {
		panic(err)
	}
	defer r.Close()

	zr, err := zlib.NewReader(r)
	if err != nil {
		panic(err)
	}
	defer zr.Close()

	buf, err = ioutil.ReadAll(zr)
	if err != nil {
		panic(err)
	}

	re, err := regexp.Compile("tree [0-9a-f]{40}")
	if err != nil {
		panic(err)
	}
	hs = re.FindString(string(buf))[5:]
	if hs == "" {
		panic("tree hash not found")
	}
	tree, err = hex.DecodeString(hs)
	if err != nil {
		panic(err)
	}

	println(tree)

	// Create several levels
	for l := 0; l < 32; l++ {

		// Create new tree object
		data = nil
		for i := 0; i < 16; i++ {
			data = append(data, []byte(fmt.Sprintf("40000 %01x\000", i))...)
			data = append(data, tree...)
		}
		tree, err = writeObject("tree", data)
		if err != nil {
			panic(err)
		}

		// Create new commit object
		data = nil
		data = append(data, []byte(fmt.Sprintf("tree %s\n", hex.EncodeToString(tree)))...)
		if commit != nil {
			data = append(data, []byte(fmt.Sprintf("parent %s\n", hex.EncodeToString(commit)))...)
		}
		data = append(data, []byte(fmt.Sprintf("author Per Persson <md2perpe@gmail.com> 0 +0100\n"))...)
		data = append(data, []byte(fmt.Sprintf("committer Per Persson <md2perpe@gmail.com> 0 +0100\n"))...)
		data = append(data, []byte("\n")...)
		data = append(data, []byte(fmt.Sprintf("Add level %d", l+1))...)
		commit, err = writeObject("commit", data)
		if err != nil {
			panic(err)
		}

		// Update master to point to last commit
		w, err := os.Create(".git/refs/heads/master")
		if err != nil {
			panic(err)
		}
		defer w.Close()

		_, err = io.WriteString(w, hex.EncodeToString(commit))
		if err != nil {
			panic(err)
		}

	}

}

func writeObject(what string, data []byte) (hash []byte, err error) {
	data = append([]byte(fmt.Sprintf("%s %d\000", what, len(data))), data...)

	ha := sha1.Sum(data)
	hash = ha[:]
	hs := hex.EncodeToString(hash)

	p := fmt.Sprintf(".git/objects/%s/%s", hs[:2], hs[2:])
	err = os.MkdirAll(filepath.Dir(p), os.ModeDir|0755)
	if err != nil {
		return
	}

	w, err := os.Create(p)
	if err != nil {
		return
	}

	zw := zlib.NewWriter(w)
	_, err = zw.Write(data)
	if err != nil {
		return
	}

	err = zw.Close()
	if err != nil {
		return
	}

	err = w.Close()
	if err != nil {
		return
	}

	log.Printf("Created object %s of type %s\n", p, what)

	return
}
