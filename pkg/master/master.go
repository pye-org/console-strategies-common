package master

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	asynq2 "github.com/pye-org/console-strategies-common/pkg/asynq"
	"golang.org/x/sync/errgroup"
	"net/http"
)

type TaskConfigGenerator func() []*asynq.PeriodicTaskConfig

type periodicTaskCfgProvider struct {
	generators []TaskConfigGenerator
}

func (p *periodicTaskCfgProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	var configs []*asynq.PeriodicTaskConfig
	for _, g := range p.generators {
		configs = append(configs, g()...)
	}
	return configs, nil
}

type Master struct {
	taskCfgProvider *periodicTaskCfgProvider
	taskManager     *asynq.PeriodicTaskManager
}

func (m *Master) Run(address string, mode string) error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		gin.SetMode(mode)
		engine := gin.New()
		engine.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/health"))
		engine.Use(gin.Recovery())
		engine.GET("/health", func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusOK, "OK")
		})
		return engine.Run(address)
	})

	eg.Go(func() error {
		return m.taskManager.Run()
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (m *Master) RegisterTaskConfigGenerator(generator TaskConfigGenerator) {
	m.taskCfgProvider.generators = append(m.taskCfgProvider.generators, generator)
}

var master *Master

func Init(config Config) error {
	if master != nil {
		return nil
	}

	redisConnOpt, err := asynq2.GetAsynqRedisConnectionOption(asynq2.Config(config))
	if err != nil {
		return err
	}

	provider := &periodicTaskCfgProvider{}
	manager, err := asynq.NewPeriodicTaskManager(asynq.PeriodicTaskManagerOpts{
		RedisConnOpt:               redisConnOpt,
		PeriodicTaskConfigProvider: provider,
	})
	if err != nil {
		return err
	}

	master = &Master{
		taskCfgProvider: provider,
		taskManager:     manager,
	}
	return nil
}

func Instance() *Master {
	return master
}
