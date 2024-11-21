package clcli

import (
	"fmt"
	"os"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/chronolens/chronolens-cli/internal/clcli/commands"
	clcli "github.com/chronolens/chronolens-cli/internal/clcli/utils"
	"golang.org/x/term"
)

var CLI struct {
	Upload struct {
		Server   string `help:"The Address of the chronolens instance"`
		Username string `help:"The username to login with"`
		Path     string `help:"The path from where to get the files to upload" type:"path"`
	} `cmd:"" help:"Bulk upload files to the chronolens instance"`
}

func Run() {
	ctx := kong.Parse(&CLI)

	fmt.Println("Please input your password")
	password_bytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("Error reading password")
	}
	password := string(password_bytes)

	api := clcli.NewAPI(CLI.Upload.Server)

	err = api.Login(CLI.Upload.Username, password)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	switch ctx.Command() {
	case "upload":
		commands.Upload(api, CLI.Upload.Path)
	default:
	}

}
