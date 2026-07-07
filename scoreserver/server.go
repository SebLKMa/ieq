package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"

	tasks "github.com/seblkma/ieq/tasks"
)

// Build and run as a binary; Ctrl-C or SIGTERM stops the tasks gracefully.
// go build -o server server.go
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for _, configFile := range []string{"configawair.yaml", "configuhoo.yaml"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task := tasks.NewScoringTask(configFile)
			log.Printf("%s task started", configFile)
			if err := task.Execute(ctx); err != nil && ctx.Err() == nil {
				log.Printf("%s task stopped: %v", configFile, err)
			}
		}()
	}

	wg.Wait() // blocks until all tasks stop (signal received)
}
