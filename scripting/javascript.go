package scripting

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/cucumber/godog"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"github.com/marmotherder/habitable/command"
	"github.com/marmotherder/habitable/common"
	"github.com/marmotherder/habitable/copy"
	"github.com/marmotherder/habitable/hashes"
)

func generateJavascriptScripts(scriptDirs []string) error {
	changes, err := hashes.CheckDirectoryHashes(scriptDirs...)
	if err != nil {
		return err
	}
	if !changes {
		return nil
	}

	for _, scriptDir := range scriptDirs {
		if _, _, err := command.RunCommand(scriptDir, "npm", "i"); err != nil {
			return err
		}
	}

	common.AppLogger.Info("scripts in defined folders have changed, creating a javascript build environment")
	buildDir := common.TempBuildDir() + "/" + "javascript"
	common.AppLogger.Trace("cleaning %s", buildDir)
	if err := os.RemoveAll(buildDir); err != nil {
		common.AppLogger.Error("failed to clean javascript build subdir")
		return err
	}

	common.AppLogger.Trace("creating %s", buildDir)
	if err := os.MkdirAll(buildDir, 0740); err != nil {
		common.AppLogger.Error("failed to create javascript build subdir")
		return err
	}

	buildFiles := map[string][]byte{
		".babelrc":      []byte(babel),
		"index.ts":      []byte(index),
		"package.json":  []byte(packageJson),
		"tsconfig.json": []byte(tsconfig),
		"webpack.js":    []byte(webpack),
	}

	for name, data := range buildFiles {
		common.AppLogger.Trace("writing file %s to directory %s", name, buildDir)
		if err := os.WriteFile(buildDir+"/"+name, data, 0740); err != nil {
			common.AppLogger.Error("failed to setup javascript build files")
			return err
		}
	}

	for idx, scriptDir := range scriptDirs {
		incBuildDir := fmt.Sprintf("%s/%d", buildDir, idx)
		common.AppLogger.Debug("copying %s to %s for build", scriptDir, incBuildDir)
		if err := os.Mkdir(incBuildDir, 0740); err != nil {
			common.AppLogger.Error("could not create directory %d to js build dir", idx)
			return err
		}
		if err := copy.CopyDirectory(scriptDir, incBuildDir); err != nil {
			common.AppLogger.Error("could not copy over directory %s to build", scriptDir)
			return err
		}
	}

	common.AppLogger.Debug("starting javascript build process")
	if _, _, err := command.RunCommand(buildDir, "npm", "i"); err != nil {
		return err
	}
	if _, _, err := command.RunCommand(buildDir, "npm", "run", "build"); err != nil {
		return err
	}

	return nil
}

const babel = `{
  "presets": [
	[
	"@babel/preset-env"
	]
  ]
}`

const index = `require("core-js/stable")
require("regenerator-runtime/runtime")`

const packageJson = `{
  "name": "scripts",
  "version": "1.0.0",
  "description": "",
  "main": "index.js",
  "scripts": {
    "build": "tsc && webpack -c ./webpack.js --mode production"
  },
  "author": "",
  "license": "GPL3",
  "dependencies": {
    "core-js": "^3.19.3",
    "regenerator-runtime": "^0.13.9"
  },
  "devDependencies": {
    "@babel/core": "^7.16.0",
    "@babel/preset-env": "^7.16.4",
    "babel-loader": "^8.2.3",
    "terser-webpack-plugin": "^5.2.5",
    "typescript": "^4.5.4",
    "webpack": "^5.65.0",
    "webpack-cli": "^4.9.1"
  }
}`

const tsconfig = `{
  "compilerOptions": {
    "alwaysStrict": true,
    "charset": "utf8",
    "declaration": true,
    "experimentalDecorators": true,
    "esModuleInterop": true,
    "lib": ["ES2021"],
    "module": "CommonJS",
    "noEmitOnError": true,
    "noFallthroughCasesInSwitch": true,
    "noImplicitAny": true,
    "noImplicitReturns": true,
    "resolveJsonModule": true,
    "strict": true,
    "stripInternal": true,
    "target": "ES2021"
  },
  "include": ["**/*.ts"],
  "exclude": ["node_modules", "lib", "__tests__", "jest.*"]
}`

const webpack = `const fs = require('fs');
const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');

let entries = [];

function walkDir(walkPath) {
  fs.readdirSync(path.resolve(walkPath)).forEach(file => {
    const absPath = path.resolve(walkPath, file);
    if (walkPath.includes('node_modules')) {
      return;
    }
    if (fs.lstatSync(absPath).isDirectory()) {
      walkDir(absPath);
    } else if (file !== 'webpack.js' && file.endsWith('.js')) {
      entries.push(absPath);
    }
  })
}

walkDir(__dirname);

module.exports = {
  entry: entries,
  module: {
    rules: [
      {
        test: /\.(js)$/,
        exclude: /node_modules/,
        use: ['babel-loader']
      }
    ]
  },
  resolve: {
    extensions: ['.js']
  },
  output: {
    path: path.resolve(__dirname, '../../scripts'),
    filename: 'scripts.js',
  },
  devServer: {
    contentBase: path.resolve(__dirname, '../'),
  },
  optimization: {
    minimizer: [new TerserPlugin(
      { 
        extractComments: false
      }
    )]
  }
};`

type javascriptScript struct {
	Path      string
	Script    string
	Context   *godog.ScenarioContext
	Habitable *Habitable
	Runtime   *goja.Runtime
}

func (j javascriptScript) getPath() string {
	return j.Path
}

func (j *javascriptScript) Load() error {
	common.AppLogger.Trace("opening script file %s", j.Path)
	script, err := os.ReadFile(j.Path)
	if err != nil {
		return err
	}

	common.AppLogger.Debug("creating javascript vm for %s", j.Path)
	vm := goja.New()

	registry := require.Registry{}
	registry.Enable(vm)

	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	j.Runtime = vm
	j.Script = string(script)

	j.Habitable.AddStep = j.AddStep

	common.AppLogger.Trace("setting global object habitable to vm for %s", j.Path)
	common.AppLogger.Debug(j.Habitable)
	if err := vm.Set("habitable", j.Habitable); err != nil {
		return err
	}

	common.AppLogger.Debug("executing %s to load plugins", j.Path)
	j.Run()

	return nil
}

func (j *javascriptScript) registerPlugin(name string, plugin interface{}) error {
	common.AppLogger.Debug("registering plugin %s to %s", name, j.Path)
	if err := j.Runtime.Set(name, plugin); err != nil {
		return err
	}

	return nil
}

func (j *javascriptScript) Run() error {
	common.AppLogger.Trace("running script %s", j.Path)
	if _, err := j.Runtime.RunString(j.Script); err != nil {
		return err
	}
	common.AppLogger.Trace("script run finished for %s", j.Path)

	return nil
}

func (j javascriptScript) AddStep(step string, function interface{}) {
	if scenarioContext == nil {
		common.AppLogger.Trace("skipping adding step %s for %s, context not yet loaded", step, j.Path)
		return
	}
	common.AppLogger.Debug("adding step %s for %s", step, j.Path)
	scenarioContext.Step(step, func(arg string) error {
		functionValue := j.Runtime.ToValue(function)
		callable, isCallable := goja.AssertFunction(functionValue)
		if !isCallable {
			return fmt.Errorf("function %s in script %s is not callable", function, j.Path)
		}

		var value goja.Value
		if intValue, err := strconv.Atoi(arg); err == nil {
			value = j.Runtime.ToValue(intValue)
		} else if boolValue, err := strconv.ParseBool(arg); err == nil {
			value = j.Runtime.ToValue(boolValue)
		} else {
			value = j.Runtime.ToValue(arg)
		}

		resp, err := callable(goja.Undefined(), value)
		if err != nil {
			return err
		}

		if !resp.SameAs(goja.Undefined()) {
			respObj := resp.ToObject(j.Runtime)
			switch respObj.ClassName() {
			case "Error":
				return errors.New(respObj.String())
			case "Promise":
				if prom, ok := resp.Export().(*goja.Promise); ok {
					prom.Result()
				}
			}
		}

		return nil
	})
}
