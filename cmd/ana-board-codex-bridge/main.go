package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/georgestander/ana-board/internal/codexbridge"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "enqueue":
		return runEnqueue(args[1:])
	case "daemon":
		return runDaemon(args[1:])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runEnqueue(args []string) error {
	config := codexbridge.DefaultConfig()
	fs := flag.NewFlagSet("enqueue", flag.ContinueOnError)
	queueDir := fs.String("queue-dir", config.QueueDir, "queue directory")
	eventFlag := fs.String("event", "", "Codex event name")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	eventName := strings.TrimSpace(*eventFlag)
	payloadArgs := fs.Args()
	if eventName == "" && len(payloadArgs) > 0 {
		eventName = payloadArgs[0]
		payloadArgs = payloadArgs[1:]
	}

	payload := []byte(strings.TrimSpace(strings.Join(payloadArgs, " ")))
	if len(payload) == 0 {
		stdinPayload, err := readOptionalStdin()
		if err != nil {
			return err
		}
		payload = stdinPayload
	}

	config.QueueDir = *queueDir
	result, err := codexbridge.Enqueue(config, eventName, payload)
	if err != nil {
		return err
	}

	if *jsonOut {
		return printJSON(result)
	}
	if result.Queued {
		fmt.Println("queued")
		return nil
	}

	fmt.Println("skipped")
	return nil
}

func runDaemon(args []string) error {
	config := codexbridge.DefaultConfig()
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	queueDir := fs.String("queue-dir", config.QueueDir, "queue directory")
	statePath := fs.String("state", config.StatePath, "state file")
	boardURL := fs.String("url", config.BoardURL, "board URL")
	source := fs.String("source", config.Source, "board message source")
	interval := fs.Duration("interval", 2*time.Second, "poll interval")
	once := fs.Bool("once", false, "process the queue once and exit")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	config.QueueDir = *queueDir
	config.StatePath = *statePath
	config.BoardURL = *boardURL
	config.Source = *source

	if *once {
		stats, err := codexbridge.ProcessOnce(context.Background(), config, nil)
		if *jsonOut {
			_ = printJSON(stats)
		}
		return err
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		stats, err := codexbridge.ProcessOnce(context.Background(), config, nil)
		if *jsonOut && (stats.Seen > 0 || err != nil) {
			_ = printJSON(stats)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		<-ticker.C
	}
}

func readOptionalStdin() ([]byte, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}
	return io.ReadAll(os.Stdin)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printHelp() {
	fmt.Println(`ana-board-codex-bridge connects Codex lifecycle events to Ana Board.

Commands:
  enqueue [--event EVENT] [EVENT] [JSON]
  daemon [--once] [--url URL] [--queue-dir DIR] [--state PATH]

Recommended Codex path:
  notify/hooks call enqueue. A LaunchAgent runs daemon in the background.

The queue stores only classified signals, not raw Codex prompt or result text.`)
}
