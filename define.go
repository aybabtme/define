package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/aybabtme/dskvs"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

const (
	dictZipFile    = "http://www.gutenberg.org/files/29765/29765-8.zip"
	dbPath         = "db"
	maxDictZipSize = 1 << 26 // ~67MB, the dict is 28MB
	maxDictSize    = 1 << 25 // ~33MB, the zip itself is 10MB

	englishDict = "en/dict"
)

func main() {

	db, err := dskvs.Open(dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dict, ok, err := db.Get(englishDict)
	if err != nil {
		panic(err)
	}

	if !ok {
		dict, err = ioutil.ReadAll(UpdateDict())
		if err != nil {
			panic(err)
		}
		db.Put(englishDict, dict)
	}

	buf := bytes.NewBuffer(dict)
	data := make([]byte, 1024)
	for i := 0; i < 1000; i++ {
		n, err := buf.Read(data)
		if err != nil {
			panic(err)
		} else if n < len(data) {
			break
		}
		fmt.Printf("%s", data)
	}
}

func UpdateDict() io.Reader {
	resp, err := http.Get(dictZipFile)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	zipBytes, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxDictZipSize))
	if err != nil {
		panic(err)
	}

	zipBuf := bytes.NewReader(zipBytes)
	zipFile, err := zip.NewReader(zipBuf, int64(zipBuf.Len()))

	if len(zipFile.File) != 1 {
		panic("Invalid file count, expected just one but was " + strconv.Itoa(len(zipFile.File)))
	}

	dictFile := zipFile.File[0]
	dict, err := dictFile.Open()
	if err != nil {
		panic(err)
	}
	defer dict.Close()

	return io.LimitReader(dict, maxDictSize)
}
