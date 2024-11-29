package commands

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/chronolens/chronolens-cli/internal/clcli/utils"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

func Upload(api clcli.API, root_dir, username string) {

	fmt.Println("Please input your password")
	password_bytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("Error reading password")
	}
	password := string(password_bytes)

	err = api.Login(username, password)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	remoteMedia, err := api.SyncFull()
	if err != nil {
		log.Fatalln(err.Error())
	}

	remoteMediaSet := map[string]struct{}{}
	for _, v := range remoteMedia {
		remoteMediaSet[v.Checksum] = struct{}{}
	}

	successful_bar := progressbar.Default(-1, "Uploading")
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

		timestamp, mimeType, err := TimestampAndMIMEType(path)
		if err != nil {
			return nil
		}

		respStatusCode, err := api.Upload(path, checksum, timestamp, mimeType)
		if err != nil {
			return nil
		}

		switch respStatusCode {
		case http.StatusOK:
			successful_bar.Add(1)
		case http.StatusPreconditionFailed:
			duplicate = duplicate + 1
		default:
			failed = failed + 1
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	successful_bar.Exit()
	fmt.Printf("Duplicate: %v\n", duplicate)
	fmt.Printf("Failed: %v\n", failed)
}

func Backup(api clcli.API, dest, username string) {

	fmt.Println("Please input your password")
	password_bytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("Error reading password")
	}
	password := string(password_bytes)

	err = api.Login(username, password)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	remoteMedia, err := api.SyncFull()
	if err != nil {
		log.Fatalln(err.Error())
	}
	backup_progress := progressbar.Default(int64(len(remoteMedia)), "Backing up")

OUTER:
	for _, v := range remoteMedia {
		fullMedia, err := api.GetFullMedia(v.Id)
		if err != nil {
			continue
		}
		created_at := time.UnixMilli(v.Timestamp)
		year, month, day := fmt.Sprintf("%v", created_at.Year()), fmt.Sprintf("%v", int(created_at.Month())), fmt.Sprintf("%v", created_at.Day())
		folderPath := filepath.Join(dest, year, month, day)
		err = os.MkdirAll(folderPath, 0750)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return
		}

		filePath := filepath.Join(folderPath, fullMedia.FileName)

		_, err = os.Stat(filePath)
		if os.IsNotExist(err) {
			clcli.DownloadFile(fullMedia.MediaURL, filePath)
			backup_progress.Add(1)
			continue
		} else {
			checksum, err := clcli.CalculateChecksums(filePath)
			if err != nil {
				backup_progress.Add(1)
				continue
			}
			if checksum == v.Checksum {
				backup_progress.Add(1)
				continue
			} else {
			INNER:
				for i := 1; ; i++ {
					ext := filepath.Ext(fullMedia.FileName)
					fileNameWithoutExt := fullMedia.FileName[:len(fullMedia.FileName)-len(ext)]
					newFileName := fmt.Sprintf("%v_%v%v", fileNameWithoutExt, i, ext)
					newFilePath := filepath.Join(folderPath, newFileName)
					_, err = os.Stat(newFilePath)
					if os.IsNotExist(err) {
						clcli.DownloadFile(fullMedia.MediaURL, newFilePath)
                        backup_progress.Add(1)
						break
					} else {
						newChecksum, err := clcli.CalculateChecksums(newFilePath)
						if err != nil {
							backup_progress.Add(1)
							continue OUTER
						}
						if newChecksum == v.Checksum {
							backup_progress.Add(1)
							continue OUTER
						} else {
							continue INNER
						}

					}

				}
			}

		}
	}

}

func CreateUser(api clcli.API, username string) {

	fmt.Println("Please input the new user's password")
	password_bytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("Error reading password")
	}
	password := string(password_bytes)

	err = api.Register(username, password)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func TimestampAndMIMEType(path string) (string, string, error) {
	fileToUpload, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer fileToUpload.Close()

	et, err := exiftool.NewExiftool()
	if err != nil {
		return "", "", err
	}
	defer et.Close()

	fileInfos := et.ExtractMetadata(path)

	if len(fileInfos) == 0 {
		return "", "", fmt.Errorf("No metadata found for file %v", path)
	}

	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		return "", "", fileInfo.Err
	}

	mime_type := fileInfo.Fields["MIMEType"]
	mime_type_string, ok := mime_type.(string)
	if !ok {
		return "", "", fmt.Errorf("No MIME type found for file %v", path)
	}

	ALLOWED_CONTENT_TYPES := []string{"image/png", "image/jpeg", "image/heic", "image/heif", "image/x-adobe-dng"}

	if !slices.Contains(ALLOWED_CONTENT_TYPES, mime_type_string) {
		return "", "", fmt.Errorf("Unsupported filetype for file %v", path)
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
