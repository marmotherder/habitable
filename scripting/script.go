package scripting

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cucumber/godog"
	"github.com/marmotherder/habitable/common"
	"github.com/marmotherder/habitable/logger"
	"github.com/marmotherder/habitable/plugins"
)

type Habitable struct {
	Logger    logger.Logger
	Variables common.HabitableVariables
	UsePlugin func(name string, version string, location ...string)
	AddStep   func(step string, function interface{})
}

type Script interface {
	getPath() string
	registerPlugin(string, interface{}) error
	Load() error
	Run() error
}

var scripts map[string]Script

func LoadScripts(dirs ...string) error {
	javascriptDirs := []string{}
	common.AppLogger.Debug("attempting to load scripts from %s", dirs)
	for _, scriptDir := range dirs {
		contents, err := os.ReadDir(scriptDir)
		if err != nil {
			common.AppLogger.Error("could not read script directory %s", scriptDir)
			continue
		}
		common.AppLogger.Debug("read directory %s for scripts", scriptDir)

		valid := false
		for _, content := range contents {
			if !content.IsDir() {
				ext := filepath.Ext(content.Name())
				if ext == ".js" {
					valid = true
					javascriptDirs = append(javascriptDirs, scriptDir)
					common.AppLogger.Debug("directory %s has scripts for javascript, adding to loader", scriptDir)
					break
				}
			}
		}

		if !valid {
			common.AppLogger.Error("script directory: %s has no supported script files", scriptDir)
		}
	}

	if err := generateJavascriptScripts(javascriptDirs); err != nil {
		common.AppLogger.Fatal(common.SetupError, err.Error())
	}

	common.AppLogger.Trace("setting up script global object")
	habitable := &Habitable{
		Logger:    common.AppLogger,
		Variables: common.Variables,
		UsePlugin: plugins.UsePlugin,
	}
	common.AppLogger.Trace(habitable)

	common.AppLogger.Trace("load process scripts in %s", common.TempScriptsDir())
	contents, err := os.ReadDir(common.TempScriptsDir())
	if err != nil {
		return err
	}

	scripts = map[string]Script{}
	for _, content := range contents {
		if !content.IsDir() {
			ext := filepath.Ext(content.Name())
			switch ext {
			case ".js", ".jsm", ".ts":
				common.AppLogger.Debug("adding processed javascript script %s to loader", content.Name())
				scripts[content.Name()] = &javascriptScript{
					Path:      fmt.Sprintf("%s/%s", common.TempScriptsDir(), content.Name()),
					Habitable: habitable,
				}
			default:
				common.AppLogger.Error("no supported file extension found for extension file: %s", content.Name())
			}
		}
	}

	for name, script := range scripts {
		common.AppLogger.Debug("loading %s", name)
		if err := script.Load(); err != nil {
			common.AppLogger.Error("failed to load script at path: %s", script.getPath())
			return err
		}
	}

	common.AppLogger.Info("resolving plugins found defined in script files")
	loadedPlugins, err := plugins.ResolvePlugins()
	if err != nil {
		return err
	}

	common.AppLogger.Info("registering plugins with scripts")
	for scriptName, script := range scripts {
		for name, plugin := range loadedPlugins {
			common.AppLogger.Debug("loading plugin %s to script %s", name, scriptName)
			if err := script.registerPlugin(name, plugin); err != nil {
				return err
			}
		}
	}

	return nil
}

var scenarioContext *godog.ScenarioContext

func RegisterSteps(ctx *godog.ScenarioContext) error {
	scenarioContext = ctx

	for name, script := range scripts {
		common.AppLogger.Debug("running %s to register defined steps", name)
		if err := script.Run(); err != nil {
			common.AppLogger.Error("failed to run script at path: %s", script.getPath())
			return err
		}
	}

	return nil
}
