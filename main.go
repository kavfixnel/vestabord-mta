package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	vestaboard "github.com/kavfixnel/vestaboard-go"
	"github.com/kavfixnel/vestaboard/mta"
)

func runSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	envPath := fs.String("env", ".env", "path to .env file")
	forced := fs.Bool("forced", false, "send even during quiet hours")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	text := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if text == "" {
		fmt.Fprintln(os.Stderr, "usage: vestaboard send [-env .env] [-forced] <message>")
		os.Exit(1)
	}

	token, err := loadEnvVar(*envPath, "VESTABORD_TOKEN")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := vestaboard.NewClient(token)
	var result *vestaboard.Message
	if *forced {
		result, err = client.WriteTextForced(text)
	} else {
		result, err = client.WriteText(text)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("sent message (id: %s)\n", result.ID)
}

func runLTrain(args []string) {
	fs := flag.NewFlagSet("l", flag.ExitOnError)
	envPath := fs.String("env", ".env", "path to .env file")
	forced := fs.Bool("forced", false, "send even during quiet hours")
	printOnly := fs.Bool("print", false, "print message without sending to Vestaboard")
	once := fs.Bool("once", false, "update once and exit")
	interval := fs.Duration("interval", 30*time.Second, "how often to refresh (e.g. 30s, 1m)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *interval < 15*time.Second && !*once {
		fmt.Fprintln(os.Stderr, "interval must be at least 15s (Vestaboard rate limit)")
		os.Exit(1)
	}

	var token string
	if !*printOnly {
		var err error
		token, err = loadEnvVar(*envPath, "VESTABORD_TOKEN")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	update := func() error {
		now := time.Now()
		arrivals, err := mta.FetchLArrivals1stAve(now)
		if err != nil {
			return err
		}

		message := mta.FormatBoardMessage(now, arrivals)
		fmt.Println(message)

		if *printOnly {
			return nil
		}

		client := vestaboard.NewClient(token)
		var msg *vestaboard.Message
		if *forced {
			msg, err = client.WriteTextForced(message)
		} else {
			msg, err = client.WriteText(message)
		}
		if err != nil {
			return err
		}

		fmt.Printf("sent message (id: %s)\n", msg.ID)
		return nil
	}

	if *once {
		if err := update(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		if err := update(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		runLTrain(nil)
		return
	}

	switch os.Args[1] {
	case "send":
		runSend(os.Args[2:])
	case "l":
		runLTrain(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `vestaboard - write to a Vestaboard Note

usage:
  vestaboard                         refresh L train arrivals every 30s
  vestaboard l [-interval 1m]        same as above
  vestaboard l -once                 update once and exit
  vestaboard l -print                preview without sending to board
  vestaboard send <msg>              send a custom message

flags:
  -env .env       path to .env file (default: .env)
  -forced         send during quiet hours
  -interval 30s   refresh frequency (minimum 15s)
  -once           update once and exit
  -print          print times without sending
`)
}
