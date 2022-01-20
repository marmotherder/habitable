package plugins

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"plugin"
	"runtime"
	"strings"

	"github.com/marmotherder/habitable/common"
	"github.com/marmotherder/habitable/copy"
	"github.com/marmotherder/habitable/hashes"
)

type HabitablePlugin interface {
	PluginObject() interface{}
}

type HabitablePluginData struct {
	Version  string
	Location string
}

var LoadPlugins map[string]HabitablePluginData

func UsePlugin(name string, version string, customLocation ...string) {
	location := "https://github.com/marmotherder/habitable-plugins/releases/download/v%s/%s_%s_%s.so"
	if len(customLocation) > 1 {
		common.AppLogger.Trace("using custom location for plugin %s", name)
		location = customLocation[0]
	}
	args := []interface{}{version, name, runtime.GOOS, runtime.GOARCH}
	n := strings.Count(location, "%s")
	if n > 0 {
		location = fmt.Sprintf(location, args[:n]...)
	}

	common.AppLogger.Debug("adding plugin %s to load from %s", name, location)

	if LoadPlugins == nil {
		LoadPlugins = make(map[string]HabitablePluginData)
	}

	LoadPlugins[name] = HabitablePluginData{
		Version:  version,
		Location: location,
	}
}

func ResolvePlugins() (map[string]interface{}, error) {
	loadedPlugins := map[string]interface{}{}
	for name, data := range LoadPlugins {
		common.AppLogger.Debug("Attempting to resolve plugin %s", name)
		hasChanges, err := hashes.CheckStringHash(name, data.Location)
		if err != nil {
			common.AppLogger.Error("failed to lookup hashes for plugin %s", name)
			return nil, err
		}

		common.AppLogger.Debug("vendor any plugins not already present")
		pluginFile, err := vendorPlugins(name, data.Location, hasChanges)
		if err != nil {
			return nil, err
		}

		common.AppLogger.Info("loading plugin %s from %s", name, pluginFile)
		loaded, err := plugin.Open(pluginFile)
		if err != nil {
			common.AppLogger.Error("failed to load plugin %s", name)
			return nil, err
		}
		entry, err := loaded.Lookup("NewPluginObject")
		if err != nil {
			common.AppLogger.Error("failed to find plugin entrypoint %s", name)
			return nil, err
		}
		entryFn, ok := entry.(func() interface{})
		if !ok {
			common.AppLogger.Error("failed to load plugin entrypoint for %s", name)
			return nil, errors.New("type did not match 'func() interface{}'")
		}

		common.AppLogger.Info("adding plugin %s to script registration loader", name)
		loadedPlugins[name] = entryFn()
	}

	return loadedPlugins, nil
}

func vendorPlugins(name, location string, vendor bool) (string, error) {
	pluginFile := ""
	if _, err := url.ParseRequestURI(location); err == nil {
		pluginFile = common.TempPluginsDir() + "/" + name + ".so"
		if !vendor {
			return pluginFile, nil
		}

		common.AppLogger.Info("downloading plugin %s from %s", name, location)
		resp, err := http.Get(location)
		if err != nil {
			common.AppLogger.Error("failed to resolve plugin for %s", name)
			return pluginFile, err
		}
		defer resp.Body.Close()

		out, err := os.Create(pluginFile)
		if err != nil {
			common.AppLogger.Error("failed to create plugin file for %s", name)
			return pluginFile, err
		}
		defer out.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			common.AppLogger.Error("failed to download plugin file for %s", name)
			return pluginFile, err
		}
		common.AppLogger.Info("successfully downloaded plugin %s to %s", name, pluginFile)
	} else {
		pluginFile = common.TempPluginsDir() + "/" + name + ".so"
		if !vendor {
			return pluginFile, nil
		}

		common.AppLogger.Info("copying plugin %s from %s", name, location)
		if err := copy.Copy(location, pluginFile); err != nil {
			common.AppLogger.Error("failed to copy plugin file for %s", name)
			return pluginFile, err
		}
		common.AppLogger.Info("successfully copied plugin %s to %s", name, pluginFile)
	}

	return pluginFile, nil
}
