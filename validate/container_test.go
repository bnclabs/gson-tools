// +build ignore

package main

import "testing"
import "os"
import "io/ioutil"
import "compress/gzip"
import "strings"
import "sort"

import "github.com/prataprc/gson"

func TestSgmtsSort(t *testing.T) {
	txt := string(testdataFile("../../testdata/typical_pointers"))
	config := gson.NewDefaultConfig()
	ptrs := make(jptrs, 0)
	for _, path := range strings.Split(txt, "\n") {
		ptrs = append(ptrs, config.NewJsonpointer(path))
	}

	sort.Sort(ptrs)
	ln := len(ptrs[0].Segments())
	for _, ptr := range ptrs[1:] {
		if len(ptr.Segments()) < ln {
			t.Errorf("sort failure")
		}
	}
}

func TestFilterAppend(t *testing.T) {
	txt := string(testdataFile("../../testdata/typical_pointers"))
	config := gson.NewDefaultConfig()
	ptrs := make(jptrs, 0)
	for _, path := range strings.Split(txt, "\n") {
		ptrs = append(ptrs, config.NewJsonpointer(path))
	}

	ptrs = ptrs.filterAppend()
	for _, ptr := range ptrs {
		segments := ptr.Segments()
		if ln := len(segments); ln > 1 && segments[ln-1][0] == '-' {
			t.Errorf("unexpected %v", ptr.Path())
		}
	}
}

func testdataFile(filename string) []byte {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var data []byte
	if strings.HasSuffix(filename, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			panic(err)
		}
		data, err = ioutil.ReadAll(gz)
		if err != nil {
			panic(err)
		}
	} else {
		data, err = ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}
	}
	return data
}
