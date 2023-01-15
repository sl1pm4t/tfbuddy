package hooks

import (
	"github.com/heptiolabs/healthcheck"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/zapier/tfbuddy/pkg/github"
	"github.com/zapier/tfbuddy/pkg/hooks_stream"
	"github.com/ziflex/lecho/v3"

	ghHooks "github.com/zapier/tfbuddy/pkg/github/hooks"
	"github.com/zapier/tfbuddy/pkg/gitlab"
	"github.com/zapier/tfbuddy/pkg/gitlab_hooks"
	tfnats "github.com/zapier/tfbuddy/pkg/nats"
	"github.com/zapier/tfbuddy/pkg/runstream"
	"github.com/zapier/tfbuddy/pkg/tfc_api"
	"github.com/zapier/tfbuddy/pkg/tfc_hooks"
)

func StartServer(worker bool, server bool) {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Logger = lecho.New(log.Logger)
	// Enable metrics middleware
	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	// add routes
	health := healthcheck.NewHandler()
	e.GET("/ready", echo.WrapHandler(health))
	e.GET("/live", echo.WrapHandler(health))

	// setup NATS client & streams
	nc := tfnats.Connect()
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		log.Fatal().Err(err).Msg("could not create Jetstream context")
	}

	hs := hooks_stream.NewHooksStream(nc)
	rs := runstream.NewStream(js)
	health.AddReadinessCheck("nats-connection", tfnats.HealthcheckFn(nc))
	health.AddLivenessCheck("nats-connection", tfnats.HealthcheckFn(nc))
	health.AddLivenessCheck("runstream-streams", rs.HealthCheck)
	health.AddLivenessCheck("hook-stream", hs.HealthCheck)

	// setup API clients
	gh := github.NewGithubClient()
	gl := gitlab.NewGitlabClient()
	tfc := tfc_api.NewTFCClient()

	if server {
		initHooksHandlers(e, tfc, rs, js, gh, gl)
	}

	if worker {
		closer := initWorkers(rs, tfc, gh, gl)
		defer closer()
	}

	if err := e.Start(":8080"); err != nil {
		log.Fatal().Err(err).Msg("could not start hooks server")
	}

}

func initHooksHandlers(e *echo.Echo, tfc tfc_api.ApiClient, rs runstream.StreamClient, js nats.JetStreamContext, gh *github.Client, gl *gitlab.GitlabClient) {
	log.Info().Msg("starting hooks handler")
	hooksGroup := e.Group("/hooks")
	hooksGroup.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		log.Trace().RawJSON("body", reqBody).Msg("Received hook request")
	}))
	logConfig := middleware.DefaultLoggerConfig
	hooksGroup.Use(middleware.LoggerWithConfig(logConfig))

	//
	// Github
	//
	githubHooksHandler := ghHooks.NewGithubHooksHandler(gh, tfc, rs, js)
	hooksGroup.POST("/github/events", githubHooksHandler.Handler)

	//
	// Gitlab
	//
	gitlabGroupHandler := gitlab_hooks.NewGitlabHooksHandler(gl, tfc, rs, js)
	hooksGroup.POST("/gitlab/group", gitlabGroupHandler.GroupHandler())
	hooksGroup.POST("/gitlab/project", gitlabGroupHandler.ProjectHandler())

	//
	// Terraform Cloud
	//
	hooksGroup.POST("/tfc/run_task", tfc_hooks.RunTaskHandler)
	// Run Notifications Handler
	notifHandler := tfc_hooks.NewNotificationHandler(tfc, rs)
	hooksGroup.POST("/tfc/notification", notifHandler.Handler())
}

func initWorkers(rs runstream.StreamClient, tfc tfc_api.ApiClient, gh *github.Client, gl *gitlab.GitlabClient) (closer func()) {
	log.Info().Msg("starting hooks workers")

	// Github Run Events Processor
	ghep := github.NewRunEventsWorker(gh, rs, tfc)

	// Gitlab Run Events Processor
	grsp := gitlab.NewRunStatusProcessor(gl, rs, tfc)

	return func() {
		ghep.Close()
		grsp.Close()
	}
}
