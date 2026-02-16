package main

import (
	"errors"
	"os"

	"github.com/mailerlite/mailerlite-cli/cmd"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var cliErr *sdkclient.CLIError
		if errors.As(err, &cliErr) && cmd.IsJSON() && len(cliErr.RawBody) > 0 {
			_ = output.JSON(cliErr.RawBody)
		} else {
			output.Error(err.Error())
		}
		os.Exit(1)
	}
}
