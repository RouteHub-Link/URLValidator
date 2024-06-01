package tasks

import (
	"log"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
)

type TaskServer struct {
	Config            TaskServerConfig
	DNSValidationTask *DNSValidationTask
	URLValidationTask *URLValidationTask
}

func (t *TaskServer) Serve() {
	asynqQueue := map[string]int{
		"critical": 6,
		"default":  3,
		"low":      1,
	}

	asynqQueue[t.URLValidationTask.Settings.Queue] = t.URLValidationTask.Settings.QueuePriority
	asynqQueue[t.DNSValidationTask.Settings.Queue] = t.DNSValidationTask.Settings.QueuePriority

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: t.Config.RedisAddr},
		asynq.Config{
			Concurrency: t.Config.Concurrency,
			Queues:      asynqQueue,
		},
	)

	mux := asynq.NewServeMux()

	mux.HandleFunc(t.URLValidationTask.TaskConfig.TaskName, t.URLValidationTask.HandleURLValidationTask)
	mux.HandleFunc(t.DNSValidationTask.TaskConfig.TaskName, t.DNSValidationTask.HandleDNSValidationTask)

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}

func (t *TaskServer) AsynqmonServe() {
	h := asynqmon.New(asynqmon.Options{
		RootPath:     t.Config.MonitoringPath,
		RedisConnOpt: asynq.RedisClientOpt{Addr: t.Config.RedisAddr},
	})

	http.Handle(h.RootPath()+"/", h)
	http.Handle("/", http.RedirectHandler(h.RootPath(), http.StatusFound))

	log.Printf("Monitoring server is running link: http://localhost:%s%s", t.Config.MonitoringPort, t.Config.MonitoringPath)
	log.Fatal(http.ListenAndServe(":"+t.Config.MonitoringPort, nil))
}
