package app

import (
	"fmt"
	"github.com/nwpc-oper/nwpc-message-client/commands"
	"github.com/nwpc-oper/nwpc-message-client/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"strconv"
	"time"
)

const ProductionMessageType = "production"

const productionDescription = `
Send messages for production.
Messages are send to a rabbitmq server directly or via a broker running by broker command.
`

type productionCommand struct {
	BaseCommand

	common.ProductionInfo
	common.ProductionEventStatus

	mainOptions struct {
		system         string
		stream         string
		productionType string
		productionName string

		event  string
		status string

		help bool
	}

	targetOptions
}

func (pc *productionCommand) runCommand(cmd *cobra.Command, args []string) error {
	err := pc.parseMainOptions(args)
	if pc.mainOptions.help {
		pc.printHelp()
		return nil
	}
	if err != nil {
		return fmt.Errorf("parse main options has error: %v", err)
	}

	err = pc.targetOptions.parseCommandTargetOptions(args)
	if err != nil {
		return fmt.Errorf("parser target options has error: %v", err)
	}

	data, err := pc.createProductionData(args)

	if err != nil {
		return fmt.Errorf("create production data has error: %s", err)
	}

	return pc.sendProductionMessage(data)
}

func newProductionCommand() *productionCommand {
	pc := &productionCommand{
		targetOptions: targetOptions{
			writeTimeout: 2 * time.Second,
			exchangeName: "nwpc.operation.production",
		},
	}

	productionCmd := &cobra.Command{
		Use:                "production",
		Short:              "send production messages",
		Long:               productionDescription,
		RunE:               pc.runCommand,
		DisableFlagParsing: true,
	}

	pc.cmd = productionCmd
	return pc
}

func (pc *productionCommand) parseMainOptions(args []string) error {
	flagSet := pc.generateCommandMainParser()
	err := flagSet.Parse(args)
	if err != nil {
		return fmt.Errorf("parse options has error: %s", err)
	}
	if pc.mainOptions.help {
		return nil
	}

	err = commands.CheckRequiredFlags(flagSet)
	if err != nil {
		return fmt.Errorf("check required flags has error: %v", err)
	}

	pc.fillProductionInfo()
	pc.fillProductionEventStatus()

	return nil
}

func (pc *productionCommand) generateCommandMainParser() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("main", pflag.ContinueOnError)
	flagSet.ParseErrorsWhitelist.UnknownFlags = true
	flagSet.StringVar(&pc.mainOptions.system, "system", "",
		"system name, such as grapes_gfs_gmf")
	flagSet.StringVar((*string)(&pc.mainOptions.productionType), "production-type", "",
		fmt.Sprintf("production type, such as %s", common.ProductionTypeGrib2))
	flagSet.StringVar((*string)(&pc.mainOptions.stream), "production-stream", "",
		"production stream, such as oper")
	flagSet.StringVar(&pc.mainOptions.productionName, "production-name", "",
		"production name, such as orig")

	flagSet.StringVar(&pc.mainOptions.event, "event", "",
		fmt.Sprintf("production event, such as %s", common.ProductionEventStorage))
	flagSet.StringVar(&pc.mainOptions.status, "status", string(common.Complete),
		fmt.Sprintf("event status, such as %s, %s", common.Complete, common.Aborted))

	flagSet.BoolVar(&pc.mainOptions.help, "help", false, "print usage")

	flagSet.SortFlags = false
	flagSet.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	flagSet.SetAnnotation("system", commands.RequiredOption, []string{"true"})
	flagSet.SetAnnotation("production-type", commands.RequiredOption, []string{"true"})
	flagSet.SetAnnotation("production-stream", commands.RequiredOption, []string{"true"})
	flagSet.SetAnnotation("production-name", commands.RequiredOption, []string{"true"})
	flagSet.SetAnnotation("event", commands.RequiredOption, []string{"true"})
	return flagSet
}

func (pc *productionCommand) fillProductionInfo() {
	pc.ProductionInfo = common.ProductionInfo{
		System: pc.mainOptions.system,
		Stream: common.ProductionStream(pc.mainOptions.stream),
		Type:   common.ProductionType(pc.mainOptions.productionType),
		Name:   common.ProductionName(pc.mainOptions.productionName),
	}
}

func (pc *productionCommand) fillProductionEventStatus() {
	status, _ := strconv.Atoi(pc.mainOptions.status)
	pc.ProductionEventStatus = common.ProductionEventStatus{
		Event:  common.ProductionEvent(pc.mainOptions.event),
		Status: common.EventStatus(status),
	}
}

func (pc *productionCommand) createProductionData(args []string) (interface{}, error) {
	var data interface{}
	var err error
	switch pc.ProductionInfo.Stream {
	case common.ProductionStreamOperation:
		data, err = pc.getOperationData(args)
		break
	case common.ProductionStreamEPS:
	default:
		err = fmt.Errorf("stream type is not supported: %s", pc.ProductionInfo.Stream)
	}

	return data, err
}

func (pc *productionCommand) getOperationData(
	args []string,
) (common.OperationProductionData, error) {
	generator := OperationPropertiesGenerator{}
	err := generator.parseOptions(args)
	if err != nil {
		return common.OperationProductionData{}, err
	}

	data := common.OperationProductionData{
		ProductionInfo:                pc.ProductionInfo,
		OperationProductionProperties: generator.OperationProductionProperties,
		ProductionEventStatus:         pc.ProductionEventStatus,
	}
	return data, nil
}

type OperationPropertiesGenerator struct {
	common.OperationProductionProperties
	options struct {
		startTime    string
		forecastTime string
	}
}

func (parser *OperationPropertiesGenerator) generateFlags() *pflag.FlagSet {
	operFlagSet := pflag.NewFlagSet("operation", pflag.ContinueOnError)
	operFlagSet.ParseErrorsWhitelist.UnknownFlags = true

	operFlagSet.StringVar(&parser.options.startTime, "start-time", "",
		"start time, YYYYMMDDHH")
	operFlagSet.StringVar(&parser.options.forecastTime, "forecast-time", "",
		"forecast time, FFFh, 0h, 12h, ...")
	operFlagSet.SetAnnotation("start-time", commands.RequiredOption, []string{"true"})
	operFlagSet.SetAnnotation("forecast-time", commands.RequiredOption, []string{"true"})
	return operFlagSet
}

func (parser *OperationPropertiesGenerator) parseOptions(args []string) error {
	operFlagSet := parser.generateFlags()
	err := operFlagSet.Parse(args)
	if err != nil {
		return fmt.Errorf("parse options has error: %s", err)
	}

	err = commands.CheckRequiredFlags(operFlagSet)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	startTime, err := time.Parse("2006010215", parser.options.startTime)
	if err != nil {
		return fmt.Errorf("parse start time %s has error: %v", parser.options.startTime, err)
	}
	parser.OperationProductionProperties = common.OperationProductionProperties{
		StartTime:    startTime,
		ForecastTime: parser.options.forecastTime,
	}

	return nil
}

func (pc *productionCommand) sendProductionMessage(data interface{}) error {
	message := common.EventMessage{
		App:  appName,
		Type: ProductionMessageType,
		Time: time.Now(),
		Data: data,
	}

	pc.targetOptions.routeKeyName = fmt.Sprintf(
		"%s.production.%s", pc.ProductionInfo.System, pc.ProductionInfo.Type)

	return sendToTarget(pc.targetOptions, message)
}

func (pc *productionCommand) printHelp() {
	fmt.Fprintf(os.Stderr, "%s\n", productionDescription)

	mainFlags := pc.generateCommandMainParser()
	fmt.Fprintf(os.Stderr, "Main Flags:\n")
	mainFlags.PrintDefaults()

	fmt.Fprintf(os.Stderr, "\n")
	targetFlags := pc.targetOptions.generateFlags()
	fmt.Fprintf(os.Stderr, "Target Flags:\n")
	targetFlags.PrintDefaults()

	operationProductionDescription := `
Operation Production
	Use stream oper. Systems include grapes_gfs_gmf, grapes_meso_3km and so on.`

	fmt.Fprintf(os.Stderr, "%s\n", operationProductionDescription)

	operGenerator := OperationPropertiesGenerator{}
	operFlags := operGenerator.generateFlags()
	fmt.Fprintf(os.Stderr, "\tFlags:\n")
	operFlags.PrintDefaults()
}
