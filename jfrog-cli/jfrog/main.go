package main

import (
	"github.com/codegangsta/cli"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/artifactory"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/bintray"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/missioncontrol"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/utils/cliutils"
	"os"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/xray"
)

const helpTemplate string =
`{{.HelpName}}{{if .UsageText}}
Arguments:
{{.UsageText}}
{{end}}{{if .Flags}}
Options:
	{{range .Flags}}{{.}}
	{{end}}{{end}}{{if .ArgsUsage}}
Environment Variables:
{{.ArgsUsage}}{{end}}

`

func main() {
	app := cli.NewApp()
	app.Name = "jfrog"
	app.Usage = "See https://github.com/jfrogdev/jfrog-cli-go for usage instructions."
	app.Version = cliutils.GetVersion()
	args := os.Args
	app.Commands = getCommands()
	cli.CommandHelpTemplate = helpTemplate
	app.Run(args)
}

func getCommands() []cli.Command {
	return []cli.Command{
		{
			Name:  	     cliutils.CmdArtifactory,
			Usage:       "Artifactory commands",
			Subcommands: artifactory.GetCommands(),
		},
		{
			Name:        cliutils.CmdBintray,
			Usage: 	     "Bintray commands",
			Subcommands: bintray.GetCommands(),
		},
		{
			Name:        cliutils.CmdMissionControl,
			Usage: 	     "Mission Control commands",
			Subcommands: missioncontrol.GetCommands(),
		},
		{
			Name:        cliutils.CmdXray,
			Usage: 	     "Xray commands",
			Subcommands: xray.GetCommands(),
		},
	}
}