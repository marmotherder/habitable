package main

import (
	"log"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/marmotherder/habitable/common"
)

func parseArgs() {
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		usedHelp := func() bool {
			for _, arg := range os.Args {
				if arg == "-h" || arg == "--help" || arg == "help" {
					return true
				}
			}
			return false
		}
		if usedHelp() {
			os.Exit(0)
		}
		log.Fatalln(err.Error())
	}

	common.Variables = make(common.HabitableVariables)
	for _, env := range os.Environ() {
		kv := strings.Split(env, "=")
		common.Variables[kv[0]] = kv[1]
	}
}
