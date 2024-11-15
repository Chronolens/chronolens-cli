package commands

import (
	"fmt"
	"log"

	"github.com/chronolens/chronolens-cli/internal/clcli/utils"
)

func Upload(base_url string, username string, password string, path string) {
	api := clcli.NewAPI(base_url)
	err := api.Login(username, password)
	if err != nil {
		print(err.Error())
	}

	// Iterate recursively in the directory
	// Calculate checksums
	checksum, err := clcli.CalculateChecksums(path)
	if err != nil {
		log.Fatalln(err.Error())
	}
	// err = api.SyncFull()
	// if err != nil {
	// 	log.Fatalln(err.Error())
	// }
	resp, err := api.Upload(path, checksum)
	if err != nil {
		log.Fatalln(err.Error())
	}
	fmt.Printf("Response Code: %v", resp.StatusCode)
}
