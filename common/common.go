package common

import (
	"github.com/marmotherder/habitable/logger"
)

var AppLogger logger.Logger

const (
	SetupError       = 1
	ScriptSetupError = 2
)

func TempDir() string {
	return "./.habitable"
}

func TempBuildDir() string {
	return TempDir() + "/" + "build"
}

func TempPluginsDir() string {
	return TempDir() + "/" + "plugins"
}

func TempScriptsDir() string {
	return TempDir() + "/" + "scripts"
}

var Variables HabitableVariables

type HabitableVariables map[string]string

func (v HabitableVariables) Get(key string) string {
	AppLogger.Trace("script looking up variable with key %s", key)
	if val, ok := v[key]; ok {
		return val
	}
	return ""
}

func (v HabitableVariables) Set(key, value string) {
	AppLogger.Trace("script setting variable with key %s", key)
	v[key] = value
}
