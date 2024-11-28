package clcli

import (
	"github.com/alecthomas/kong"
	"github.com/chronolens/chronolens-cli/internal/clcli/commands"
	clcli "github.com/chronolens/chronolens-cli/internal/clcli/utils"
)

var CLI struct {
	Server string `help:"The Address of the chronolens instance"`

	Upload struct {
		Username string `help:"The username to login with"`
		Path     string `help:"The path from where to get the files to upload" type:"path"`
	} `cmd:"" help:"Bulk upload files to the chronolens instance"`

	CreateUser struct {
		Username string `help:"The username of the user to create"`
	} `cmd:"" help:"Create a new user in the chronolens instance"`
}

func Run() {
	ctx := kong.Parse(&CLI)

	api := clcli.NewAPI(CLI.Server)

	switch ctx.Command() {
	case "upload":
		commands.Upload(api, CLI.Upload.Path, CLI.Upload.Username)
	case "create-user":
		commands.CreateUser(api,CLI.CreateUser.Username)
	default:
	}

}
