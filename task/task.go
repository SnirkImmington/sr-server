package task

import (
	"log"
	"os"
)

// RunSelectedTask runs the passed in task from the command line
func RunSelectedTask(task string, args []string) {
	log.Printf("Run task %v %v", task, args)
	switch task {
	case "migrate":
		if len(args) != 1 {
			log.Printf("Usage: migrate <gameID>")
			os.Exit(1)
		}
	}
}
