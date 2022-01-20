package main

import (
	"os"

	"github.com/marmotherder/habitable/common"
	"github.com/marmotherder/habitable/logger"
	"github.com/marmotherder/habitable/scripting"

	"github.com/cucumber/godog"
)

var opts struct {
	LogLevel   []bool   `short:"l" long:"loglevel" description:"Level of logging verbosity"`
	Clean      bool     `short:"c" long:"clean" description:"Clean .habitable directory before run"`
	TestFormat string   `short:"f" long:"format" description:"Test format to use" default:"junit"`
	Tests      []string `short:"t" long:"test" description:"Path to a BDD test file to run"`
	TestName   string   `short:"n" long:"name" description:"Name of the full test suite" default:"habitable"`
	ScriptDirs []string `short:"s" long:"extensions" description:"Path to custom extensions directory" default:"./_scripts"`
}

func main() {
	parseArgs()

	common.AppLogger = logger.DefaultLogger{
		Level: len(opts.LogLevel),
	}

	if opts.Clean {
		common.AppLogger.Info("running clean on .habitable")
		if err := os.RemoveAll(common.TempDir()); err != nil {
			common.AppLogger.Fatal(common.SetupError, err.Error())
		}
	}

	common.AppLogger.Info("constructing .habitable directory in current location")
	for _, path := range []string{common.TempBuildDir(), common.TempPluginsDir(), common.TempScriptsDir()} {
		if err := os.MkdirAll(path, 0740); err != nil {
			common.AppLogger.Fatal(common.SetupError, err.Error())
		}
	}

	common.AppLogger.Info("loading scripts")
	if err := scripting.LoadScripts(opts.ScriptDirs...); err != nil {
		common.AppLogger.Fatal(common.ScriptSetupError, err.Error())
	}

	godogOpts := &godog.Options{
		Paths:  opts.Tests,
		Format: opts.TestFormat,
	}

	godog.BindCommandLineFlags("godog.", godogOpts)

	status := godog.TestSuite{
		Name:                 opts.TestName,
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              godogOpts,
	}.Run()

	common.AppLogger.Info(status)
}
