package clcli

import (
	"github.com/alecthomas/kong"
	"github.com/chronolens/chronolens-cli/internal/clcli/commands"
)

var CLI struct {
	Upload struct {
		Server   string `help:"The Address of the chronolens instance"`
		Username string `help:"The username to login with"`
		Password string
		Path     string `help:"The path from where to get the files to upload" type:"path"`
	} `cmd:"" help:"Bulk upload files to the chronolens instance"`
}

func Run() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "upload":
		commands.Upload(CLI.Upload.Server, CLI.Upload.Username, CLI.Upload.Password, CLI.Upload.Path)
	default:
	}

}
