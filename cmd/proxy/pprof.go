//go:build pprof

package main

import (
	"log"
	"os"
	"runtime/pprof"
	"time"
)

const (
	RUN_TIME         = 60 * time.Second
	CPU_PROFILE_FILE = "proxy.prof"
)

func init() {
	log.Printf("CPU Profiling enabled, run time: %s, file: %s\n", RUN_TIME, CPU_PROFILE_FILE)

	file, err := os.Create(CPU_PROFILE_FILE)
	if err != nil {
		log.Fatal(err)
	}

	pprof.StartCPUProfile(file)

	go func() {
		time.Sleep(RUN_TIME)
		pprof.StopCPUProfile()
		os.Exit(0)
	}()
}
