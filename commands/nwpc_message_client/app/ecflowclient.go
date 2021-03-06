package app

import (
	"fmt"
	"github.com/nwpc-oper/nwpc-message-client/commands"
	"github.com/nwpc-oper/nwpc-message-client/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"time"
)

const EcflowClientMessageType = "ecflow-client"
const ecflowClientDescription = `
Send messages for ecflow_client command. 
Messages are send to a rabbitmq server directly or via a broker running by nwpc_message_client broker command.
`

func newEcflowClientCommand() *ecflowClientCommand {
	ec := &ecflowClientCommand{
		targetParser: targetParser{
			defaultOption: targetOptions{
				writeTimeout: 2 * time.Second,
				useBroker:    true,
				brokerTries:  2,
				exchangeName: "nwpc.operation.workflow",
				routeKeyName: "ecflow.command.ecflow_client",
			},
		},
	}
	ecFlowClientCmd := &cobra.Command{
		Use:                "ecflow-client",
		Short:              "send messages for ecflow_client command",
		Long:               ecflowClientDescription,
		RunE:               ec.runCommand,
		DisableFlagParsing: true,
	}
	ecFlowClientCmd.SetUsageFunc(func(*cobra.Command) error {
		ec.printHelp()
		return nil
	})
	ecFlowClientCmd.SetHelpFunc(func(*cobra.Command, []string) {
		ec.printHelp()
	})

	ec.cmd = ecFlowClientCmd
	return ec
}

type ecflowClientCommand struct {
	BaseCommand

	mainOptions struct {
		commandOptions string
		help           bool
	}

	targetParser
}

func (ec *ecflowClientCommand) runCommand(cmd *cobra.Command, args []string) error {
	err := ec.parseMainOptions(args)
	if err != nil {
		return fmt.Errorf("parse main options has eror: %v", err)
	}
	if ec.mainOptions.help {
		ec.printHelp()
		return nil
	}

	err = ec.targetParser.parseCommandTargetOptions(args)
	if err != nil {
		return fmt.Errorf("parse target options has eror: %v", err)
	}

	data, err := common.CreateEcflowClientMessage(ec.mainOptions.commandOptions)
	if err != nil {
		return err
	}

	message := common.EventMessage{
		App:  appName,
		Type: EcflowClientMessageType,
		Time: time.Now(),
		Data: data,
	}

	return sendEventMessageToTarget(ec.targetParser.option, message)
}

func (ec *ecflowClientCommand) generateMainFlags() *pflag.FlagSet {
	mainFlagSet := pflag.NewFlagSet("main", pflag.ContinueOnError)
	mainFlagSet.ParseErrorsWhitelist.UnknownFlags = true

	mainFlagSet.StringVar(&ec.mainOptions.commandOptions, "command-options", "",
		"ecflow_client command options, such as "+
			"--host=login_a06 --port=33083 "+
			"--alter add variable ECF_RID 16934800 "+
			"/gmf_grapes_gfs_post/00/togrib2/togrib2_gfs/045/ne_grib2_045")
	mainFlagSet.BoolVar(&ec.mainOptions.help, "help", false, "print usage")

	mainFlagSet.SetAnnotation("command-options", commands.RequiredOption, []string{"true"})

	return mainFlagSet
}

func (ec *ecflowClientCommand) parseMainOptions(args []string) error {
	mainFlagSet := ec.generateMainFlags()
	err := mainFlagSet.Parse(args)
	if err != nil {
		return fmt.Errorf("parse options has error: %s", err)
	}
	if ec.mainOptions.help {
		return nil
	}

	err = commands.CheckRequiredFlags(mainFlagSet)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func (ec *ecflowClientCommand) printHelp() {
	helpOutput := os.Stdout
	fmt.Fprintf(helpOutput, "%s\n", ecflowClientDescription)

	mainFlags := ec.generateMainFlags()
	mainFlags.SetOutput(helpOutput)
	fmt.Fprintf(helpOutput, "Main Flags:\n")
	mainFlags.PrintDefaults()

	fmt.Fprintf(helpOutput, "\n")
	targetFlags := ec.targetParser.generateFlags()
	targetFlags.SetOutput(helpOutput)
	fmt.Fprintf(helpOutput, "Target Flags:\n")
	targetFlags.PrintDefaults()
}
