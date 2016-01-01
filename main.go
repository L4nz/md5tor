package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/marksamman/bencode"
)

func halt(message string) {
	fmt.Println(message)
	os.Exit(1)
}

func md5sum(fileName string) (string, error) {
	fileHandle, err := os.Open(fileName)
	if err != nil {
		return "", err
	}

	defer fileHandle.Close()
	buf := md5.New()
	io.Copy(buf, fileHandle)
	sum := hex.EncodeToString(buf.Sum(nil))
	return sum, nil
}

func main() {
	if len(os.Args) < 2 {
		halt("md5tor <torrent> <directory>")
	}
	directory := filepath.Clean(os.Args[2])
	torrentFileName := filepath.Clean(os.Args[1])
	torrentFile, err := os.Open(torrentFileName)
	if err != nil {
		halt(err.Error())
	}

	dict, err := bencode.Decode(torrentFile)
	if err != nil {
		halt(err.Error())
	}
	torrentFile.Close()

	info := dict["info"].(map[string]interface{})
	writeTorrent := true

	// NOTE(L4nz): the torrent spec handles single file and multi file torrents
	// differently so we need to handle both cases.
	if info["files"] != nil {
		files := info["files"].([]interface{})
		fmt.Printf("Generating md5 sums for %d files...\n", len(files))
		for _, content := range files {
			// NOTE(L4nz): The path to a file is stored as a list so we need to join
			// each part together first.
			file := content.(map[string]interface{})
			var fileName string
			for _, content := range file["path"].([]string) {
				fileName = filepath.Join(fileName, content)
			}
			filePath := filepath.Join(directory, fileName)

			md5sum, err := md5sum(filePath)
			if err != nil {
				halt(fmt.Sprintf("Error, could not read file: %s.\n", filePath))
			}

			if file["md5sum"] != nil {
				if strings.ToLower(file["md5sum"].(string)) == md5sum {
					fmt.Printf("%s  %s [MATCH]\n", md5sum, fileName)
				} else {
					fmt.Printf("%s  %s [WRONG]\n", md5sum, fileName)
				}
				writeTorrent = false
			} else {
				fmt.Printf("%s  %s\n", md5sum, fileName)
				file["md5sum"] = md5sum
				writeTorrent = true
			}
		}

		fmt.Printf("Done. %d files processed.\n", len(files))
	} else if info["name"] != nil {
		fmt.Println("Generating md5 sum for single file torrent...")
		path := info["name"].(string)
		fileName := filepath.Join(directory, path)
		md5sum, err := md5sum(fileName)
		if err != nil {
			halt(fmt.Sprintf("error, could not read file: %s.\n", fileName))
		}

		if info["md5sum"] != nil {
			if info["md5sum"] == md5sum {
				fmt.Printf("%s  %s [MATCH]\n", md5sum, fileName)
			} else {
				fmt.Printf("%s  %s [WRONG]\n", md5sum, fileName)
			}
			writeTorrent = false
		} else {
			fmt.Printf("%s  %s\n", md5sum, fileName)
			info["md5sum"] = md5sum
		}

		fmt.Println("Done. 1 file processed.")
	} else {
		halt("Error, the torrent must contain at least one file.")
	}

	if writeTorrent {
		fmt.Println("Writing torrent file.")
		torrentFile, err = os.Create(torrentFileName)
		if err != nil {
			halt(err.Error())
		}
		torrentFile.Write(bencode.Encode(dict))
		torrentFile.Close()
	} else {
		fmt.Println("Torrent already contains md5 sums, not writing to torrent file.")
	}
}
