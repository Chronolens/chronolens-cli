package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/chronolens/chronolens-cli/internal/clcli/utils"
	"github.com/schollz/progressbar/v3"
)

func Upload(base_url string, username string, password string, root_dir string) {
	api := clcli.NewAPI(base_url)
	err := api.Login(username, password)
	if err != nil {
		print(err.Error())
	}

	remoteMedia, err := api.SyncFull()
	if err != nil {
		log.Fatalln(err.Error())
	}

	remoteMediaSet := map[string]struct{}{}
	for _, v := range remoteMedia {
		remoteMediaSet[v.Checksum] = struct{}{}
	}

	successful_bar := progressbar.Default(-1, "Successful")
	duplicate := 0
	failed := 0

	err = filepath.WalkDir(root_dir, func(path string, d os.DirEntry, err error) error {

		if d.IsDir() {
			return nil
		}

		checksum, err := clcli.CalculateChecksums(path)
		if err != nil {
			return nil
		}

		// Check if already exists
		_, ok := remoteMediaSet[checksum]
		if ok {
			return nil
		}

        timestamp,mimeType,err := TimestampAndMIMEType(path, ok)
        if err != nil {
            return err
        }

		resp, err := api.Upload(path, checksum,timestamp,mimeType)
		if err != nil {
			return nil
		}

		switch resp.StatusCode {
		case 200:
			successful_bar.Add(1)
		case 412:
			duplicate = duplicate + 1
		default:
			failed = failed + 1
		}
        time.Sleep(1 * time.Second)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nDuplicate: %v\n", duplicate)
	fmt.Printf("Failed: %v\n", failed)
}

func TimestampAndMIMEType(path string, ok bool) (string,string, error) {
	fileToUpload, err := os.Open(path)
	if err != nil {
        return "", "", nil
	}
	defer fileToUpload.Close()

	et, err := exiftool.NewExiftool()
	if err != nil {
        return "", "", nil
	}
	defer et.Close()

	fileInfos := et.ExtractMetadata(path)

	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
        return "", "", nil
	}

	mime_type := fileInfo.Fields["MIMEType"]
    mime_type_string, ok := mime_type.(string)
	if !ok {
        return "", "", nil
	}

    timestamp := getEXIFTimestamp(fileInfo)
    
	return fmt.Sprintf("%v", timestamp), mime_type_string, nil
}

func getEXIFTimestamp(fileInfo exiftool.FileMetadata) int64 {
    // Layouts for parsing
	layoutWithSubSec := "2006:01:02 15:04:05.999999"
	layoutNoSubSec := "2006:01:02 15:04:05"
	layoutWithTZ := "2006:01:02 15:04:05-07:00"

	// Function to parse a string into Unix millis
	parseToUnixMillis := func(value interface{}, layout string) int64 {
		if value != nil {
			parsedTime, err := time.Parse(layout, value.(string))
			if err == nil {
				return parsedTime.UnixMilli()
			}
		}
		return 0
	}

	// Chain of fields to check
	fields := []struct {
		key    string
		layout string
	}{
		{"SubSecDateTimeOriginal", layoutWithSubSec},
		{"DateTimeOriginal", layoutNoSubSec},
		{"SubSecCreateDate", layoutWithSubSec},
		{"CreateDate", layoutNoSubSec},
		{"FileModifyDate", layoutWithTZ},
		{"FileInodeModifyDate", layoutNoSubSec},
	}

	// Iterate through fields and return the first valid Unix millis
	for _, field := range fields {
		unixMillis := parseToUnixMillis(fileInfo.Fields[field.key], field.layout)
		if unixMillis > 0 {
			return unixMillis
		}
	}
    return time.Now().UTC().UnixMilli()
}
