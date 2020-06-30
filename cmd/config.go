package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/superfly/flyctl/cmdctx"

	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/superfly/flyctl/api"
	"github.com/superfly/flyctl/docstrings"
	"github.com/superfly/flyctl/flyctl"
	"github.com/superfly/flyctl/helpers"
)

func newConfigCommand() *Command {

	configStrings := docstrings.Get("config")

	cmd := &Command{
		Command: &cobra.Command{
			Use:   configStrings.Usage,
			Short: configStrings.Short,
			Long:  configStrings.Long,
		},
	}

	configDisplayStrings := docstrings.Get("config.display")
	BuildCommand(cmd, runDisplayConfig, configDisplayStrings.Usage, configDisplayStrings.Short, configDisplayStrings.Long, os.Stdout, requireSession, requireAppName)

	configSaveStrings := docstrings.Get("config.save")
	BuildCommand(cmd, runSaveConfig, configSaveStrings.Usage, configSaveStrings.Short, configSaveStrings.Long, os.Stdout, requireSession, requireAppName)

	configValidateStrings := docstrings.Get("config.validate")
	BuildCommand(cmd, runValidateConfig, configValidateStrings.Usage, configValidateStrings.Short, configValidateStrings.Long, os.Stdout, requireSession, requireAppName)

	return cmd
}

func runDisplayConfig(ctx *cmdctx.CmdContext) error {
	cfg, err := ctx.Client.API().GetConfig(ctx.AppName)
	if err != nil {
		return err
	}

	if ctx.OutputJSON() {
		ctx.WriteJSON(cfg.Definition)
		return nil
	}

	printValue(0, "Services", cfg.Definition)

	return nil
}

func keyFormat(key string) string {
	newkey := strings.ReplaceAll(key, "_", " ")
	return strings.Title(newkey)
}

func printValue(depth int, key string, value interface{}) {
	//fmt.Printf("%T %d\n", value, depth)
	indent := strings.Repeat(" ", depth*2)
	switch v := value.(type) {
	case string:
		fmt.Printf("%s %s: %s\n", indent, keyFormat(key), v)
	case int:
		fmt.Printf("%s %s: %d\n", indent, keyFormat(key), v)
	case float64:
		fmt.Printf("%s %s: %d\n", indent, keyFormat(key), int(v))
	case map[string]interface{}:
		if key != "" {
			fmt.Printf("\n%s %s\n", indent, keyFormat(key))
		}
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			printValue(depth+1, k, v[k])
		}
		fmt.Println()
	case []interface{}:
		if key == "handlers" {
			s := make([]string, len(v))
			for z, v := range v {
				s[z] = fmt.Sprint(v)
			}
			fmt.Printf("%s %s: [ %s ]\n", indent, keyFormat(key), strings.Join(s, ","))
		} else {
			if key != "" {
				fmt.Printf("\n%s %s\n", indent, keyFormat(key))
			}
			for _, v2 := range v {
				printValue(depth+1, "", v2)
			}
		}
	case api.Definition:
		fmt.Println(keyFormat(key))
		for _, s := range v {
			printValue(0, "", s)
		}
	default:
		fmt.Println("Nope")
		fmt.Printf("%+v %T\n", v, v)
	}
}

func runSaveConfig(ctx *cmdctx.CmdContext) error {
	configfilename, err := flyctl.ResolveConfigFileFromPath(ctx.WorkingDir)

	if helpers.FileExists(configfilename) {
		ctx.Status("create", cmdctx.SERROR, "An existing configuration file has been found.")
		confirmation := confirm(fmt.Sprintf("Overwrite file '%s'", configfilename))
		if !confirmation {
			return nil
		}
	}

	if ctx.AppConfig == nil {
		ctx.AppConfig = flyctl.NewAppConfig()
	}
	ctx.AppConfig.AppName = ctx.AppName

	serverCfg, err := ctx.Client.API().GetConfig(ctx.AppName)
	if err != nil {
		return err
	}

	ctx.AppConfig.Definition = serverCfg.Definition

	return writeAppConfig(ctx.ConfigFile, ctx.AppConfig)
}

func runValidateConfig(commandContext *cmdctx.CmdContext) error {
	if commandContext.AppConfig == nil {
		return errors.New("App config file not found")
	}

	commandContext.Status("flyctl", cmdctx.STITLE, "Validating", commandContext.ConfigFile)

	serverCfg, err := commandContext.Client.API().ParseConfig(commandContext.AppName, commandContext.AppConfig.Definition)
	if err != nil {
		return err
	}

	if serverCfg.Valid {
		fmt.Println(aurora.Green("✓").String(), "Configuration is valid")
		return nil
	}

	printAppConfigErrors(*serverCfg)

	return errors.New("App configuration is not valid")
}

func printAppConfigErrors(cfg api.AppConfig) {
	fmt.Println()
	for _, error := range cfg.Errors {
		fmt.Println("   ", aurora.Red("✘").String(), error)
	}
	fmt.Println()
}

func writeAppConfig(path string, appConfig *flyctl.AppConfig) error {

	if err := appConfig.WriteToFile(path); err != nil {
		return err
	}

	fmt.Println("Wrote config file", helpers.PathRelativeToCWD(path))

	return nil
}
