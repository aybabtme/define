package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/aybabtme/dskvs"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	dictZipFile       = "http://www.gutenberg.org/files/29765/29765-8.zip"
	dbPath            = "db"
	maxDictZipSize    = 1 << 26 // ~67MB, the dict is 28MB
	maxDictSize       = 1 << 25 // ~33MB, the zip itself is 10MB
	downloadBlockSize = 1 << 13 // 8k, size of a packet
	englishDict       = "en/dict"
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
		dict, err = UpdateDict()
		if err != nil {
			panic(err)
		}
		db.Put(englishDict, dict)
	}

	// TODO: Do something with the dict (parse it for definitions)
}

func UpdateDict() ([]byte, error) {
	fmt.Printf("Updating dictionary from %s\n", dictZipFile)

	resp, err := http.Get(dictZipFile)
	if err != nil {
		return nil, fmt.Errorf("getting %s, %v", dictZipFile, err)
	}
	defer resp.Body.Close()

	bodyRdr := io.LimitReader(resp.Body, maxDictZipSize)
	progressUpdt := GetProgressFunc(resp.ContentLength)

	zipBytes, err := DownloadFile(bodyRdr, resp.ContentLength, progressUpdt)
	if err != nil {
		return nil, fmt.Errorf("reading content from limited reader of size %d, %v", maxDictZipSize, err)
	}

	fmt.Println("Unzipping")

	zipBuf := bytes.NewReader(zipBytes)
	zipFile, err := zip.NewReader(zipBuf, int64(zipBuf.Len()))

	if len(zipFile.File) != 1 {
		return nil, fmt.Errorf("invalid file count, expected just one but was %d", len(zipFile.File))
	}

	fmt.Println("Reading zipped dictionary file")
	dictFile := zipFile.File[0]
	dict, err := dictFile.Open()
	if err != nil {
		return nil, fmt.Errorf("opening dict file from zip archive, %v", err)
	}
	defer dict.Close()

	dictData, err := ioutil.ReadAll(io.LimitReader(dict, maxDictSize))

	fmt.Println("Done")
	return dictData, err
}

func GetProgressFunc(total int64) func(int64) {
	return func(i int64) {
		percDone := float64(i) / float64(total) * 100.0
		fmt.Printf("%3.2f percent done, %d/%d bytes\r", percDone, i, total)
	}
}

func DownloadFile(r io.Reader, totalSize int64, progressUpdt func(i int64)) ([]byte, error) {
	out := bytes.NewBuffer(make([]byte, 0, totalSize))
	byteRead := int64(0)

	fmt.Println("Download starts")
	for {

		n, err := io.CopyN(out, r, downloadBlockSize)

		byteRead += n
		progressUpdt(byteRead)

		if n < downloadBlockSize {
			break
		} else if err != nil {
			return nil, err
		}

	}
	fmt.Println("\nDownload done")

	return out.Bytes(), nil
}
