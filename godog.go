package main

import (
	"context"

	"github.com/cucumber/godog"
	"github.com/hoisie/mustache"
	"github.com/marmotherder/habitable/common"
	"github.com/marmotherder/habitable/scripting"
)

func InitializeTestSuite(ctx *godog.TestSuiteContext) {}

func InitializeScenario(ctx *godog.ScenarioContext) {
	common.AppLogger.Debug("running godog with context %s", *ctx)
	ctx.StepContext().Before(func(ctx context.Context, st *godog.Step) (context.Context, error) {
		common.AppLogger.Debug("performing feature file substitution")
		mustacheText, err := mustache.ParseString(st.Text)
		if err != nil {
			return nil, err
		}
		st.Text = mustacheText.Render(common.Variables)
		common.AppLogger.Trace(st.Text)
		return ctx, nil
	})

	common.AppLogger.Info("registering script defined steps")
	if err := scripting.RegisterSteps(ctx); err != nil {
		common.AppLogger.Fatal(common.ScriptSetupError, err.Error())
	}
}
