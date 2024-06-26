package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/RouteHub-Link/DomainUtils/tasks"
	tasks_config "github.com/RouteHub-Link/DomainUtils/tasks/config"
	"github.com/RouteHub-Link/DomainUtils/validator"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
	"gopkg.in/redis.v5"

	"dario.cat/mergo"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	flag "github.com/spf13/pflag"
)

var (
	_appConfig = &ApplicationConfig{
		ValidatorConfig:  validator.CheckConfig{},
		TaskServerConfig: tasks.TaskServerConfig{},
		TaskConfigs: tasks_config.TaskConfigs{
			DNSValidation:  tasks_config.DefaultDNSValidationTaskConfig,
			URLValidation:  tasks_config.DefaultURLValidationTaskConfig,
			SiteValidation: tasks_config.DefaultSiteValidationTaskConfig,
		},
	}

	onceConfigure sync.Once
	conf          = koanf.Conf{
		Delim:       ".",
		StrictMerge: true,
	}
	k = koanf.NewWithConf(conf)
)

func GetApplicationConfig() *ApplicationConfig {
	onceConfigure.Do(func() {

		setDefaults()

		loadConfigYaml()

		loadEnv()

		parseFlags()

		checkRedis()

		log.Printf("Application Config : %+v", _appConfig)
	})

	return _appConfig
}

func loadConfigYaml() {
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		log.Printf("error loading config: %v", err)
	}

	err := k.Unmarshal("", _appConfig)
	if err != nil {
		log.Printf("error unmarshal config: %v", err)
	}
}

func loadEnv() {
	k.Load(env.Provider("", ".", func(s string) string {
		return s
	}), nil)

	err := k.Unmarshal("", _appConfig)
	if err != nil {
		log.Printf("error unmarshal config: %v", err)
	}
}

func checkRedis() {
	log.Printf("Redis Config : %s", _appConfig.TaskServerConfig.RedisAddr)

	if _appConfig.TaskServerConfig.RedisAddr == "" {
		log.Fatalf("error loading redis address from config")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: _appConfig.TaskServerConfig.RedisAddr,
	})

	res, err := redisClient.Ping().Result()
	if err != nil && res != "PONG" {
		log.Fatalf("error connecting redis Client %+v", err)
	}
}

func setDefaults() {
	if err := mergo.Merge(&_appConfig.ValidatorConfig, validator.DefaultCheckConfig); err != nil {
		log.Fatalf("error merging validator config: %v", err)
	}

	if err := mergo.Merge(&_appConfig.TaskServerConfig, tasks.DefaultTaskServerConfig, mergo.WithoutDereference); err != nil {
		log.Fatalf("error merging task server config: %v", err)
	}

	if err := mergo.Merge(&_appConfig.TaskConfigs.DNSValidation, tasks_config.DefaultDNSValidationTaskConfig); err != nil {
		log.Fatalf("error merging dns validation task config: %v", err)
	}

	if err := mergo.Merge(&_appConfig.TaskConfigs.URLValidation, tasks_config.DefaultURLValidationTaskConfig); err != nil {
		log.Fatalf("error merging url validation task config: %v", err)
	}

	if err := mergo.Merge(&_appConfig.TaskConfigs.SiteValidation, tasks_config.DefaultSiteValidationTaskConfig); err != nil {
		log.Fatalf("error merging site validation task config: %v", err)
	}

	if _appConfig.Port == "" {
		log.Printf("error loading port from config setting default port 8080")
		_appConfig.Port = "8080"
	}
}

func parseFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	f.StringVarP(&_appConfig.Port, "receiver port", "p", _appConfig.Port, "Port to listen on")
	f.StringVarP(&_appConfig.TaskServerConfig.MonitoringPort, "monitoring port", "w", _appConfig.TaskServerConfig.MonitoringPort, "Port to listen on for monitoring dashboard")

	f.BoolVarP(&_appConfig.Health, "health", "h", _appConfig.Health, "Enable health check endpoint")

	f.StringVarP(&_appConfig.TaskServerConfig.RedisAddr, "redis-addr", "r", _appConfig.TaskServerConfig.RedisAddr, "Redis address")

	f.BoolVarP(&_appConfig.TaskServerConfig.MonitoringDash, "monitoring-dash", "m", _appConfig.TaskServerConfig.MonitoringDash, "Enable monitoring dashboard")

	var hostingMode int8
	f.Int8VarP(&hostingMode, "serving-mode", "s", hostingMode, "Serves application as selected mode.\n-s 0 or --serving-mode 0 \n0 : TaskReceiver\n1 : TaskServer\n2 : TaskMonitoring\nDefault : 0")

	f.Parse(os.Args[1:])

	if hostingMode != 0 && _appConfig.HostingMode.Get().Id() != 0 {
		_appConfig.HostingMode = HostingMode(hostingMode)
	}
}
