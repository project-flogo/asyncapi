package cmd

import (
	"github.com/project-flogo/cli/common"
	"github.com/spf13/cobra"

	"github.com/project-flogo/asyncapi/transform"
)

func init() {
	appgen.Flags().StringVarP(&input, "input", "i", "asyncapi.yml", "path to input swagger file")
	appgen.Flags().StringVarP(&conversionType, "type", "t", "flogoapiapp", "conversion type like flogoapiapp or flogodescriptor")
	appgen.Flags().StringVarP(&output, "output", "o", ".", "path to generated file")
	common.RegisterPlugin(appgen)
}

var input, conversionType, output string
var appgen = &cobra.Command{
	Use:              "asyncapi",
	Short:            "generates flogo app",
	Long:             "generates flogo application for supplied async api specification",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		transform.Transform(input, output, conversionType)
	},
}
