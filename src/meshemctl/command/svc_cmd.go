package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/rerorero/meshem/src/model"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	filePath string
)

// NewServiceCommand returns the command object for 'svc'.
func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "svc <subcommand>",
		Short: "Service related commands",
	}
	cmd.AddCommand(newApplyServiceCommand())
	return cmd
}

func newApplyServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <servicename> -f <filename>",
		Short: "Apply a configuration to a service by filename",
		Run:   applyService,
	}
	cmd.Flags().StringVarP(&filePath, "filepath", "f", "", "(required) File path that defines the service")
	return cmd
}

func applyService(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(errors.New("command needs an argument as service name"))
	}
	serviceName := args[0]
	if len(filePath) == 0 {
		ExitWithError(fmt.Errorf("command needs --filepath argument"))
	}

	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		ExitWithError(errors.Wrapf(err, "could not read resource file(%s)", filePath))
	}

	var param model.IdempotentServiceParam
	err = yaml.Unmarshal(buf, &param)
	if err != nil {
		ExitWithError(errors.Wrapf(err, "failed to parse resource file(%s)", filePath))
	}

	client, err := NewAPIClient()
	if err != nil {
		ExitWithError(err)
	}

	resp, _, err := client.PutService(serviceName, param)
	if err != nil {
		ExitWithError(err)
	}

	fmt.Printf("OK (Changed=%t)\n", resp.Changed)
}

func showService(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(errors.New("command needs an argument as service name"))
	}
	serviceName := args[0]

	client, err := NewAPIClient()
	if err != nil {
		ExitWithError(err)
	}

	resp, _, err := client.GetService(serviceName)
	if err != nil {
		ExitWithError(err)
	}
	byte, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		ExitWithError(errors.Wrapf(err, "failed to parse the response as JSON: %+v", resp))
	}

	fmt.Println(string(byte))
}
