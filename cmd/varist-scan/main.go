package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
	"gopkg.intern.drachenfels.de/drachenfels/varist-tools-ha/internal/parser"
)

func main() {
	senderIP := flag.String("senderip", "", "Sender IP address")
	mailFrom := flag.String("mailfrom", "", "MAIL FROM address")
	msgID := flag.String("msgid", "", "Message ID")
	scanFile := flag.String("scanfile", "", "Path to scan file")
	configPath := flag.String("config-file", getEnv("VARIST_CONFIG_FILE", "/workdir/userconf/confremote/varistav.conf"), "Path to config INI file")
	responseFile := flag.String("response-file", getEnv("VARIST_RESPONSE_FILE", "/workdir/workspace/var/spool/exim4/hybrid-analyzer/response.log"), "Path to response log file")
	timeoutSeconds := flag.Int("ha-timeout", getEnvAsInt("VARIST_HA_TIMEOUT", 10), "HA HTTP request timeout in seconds")
	verbose := flag.Bool("v", false, "Verbose output")
	rating := flag.Float64("rating", 0, "Minimum rating to include")
	flag.Parse()

	if *senderIP == "" || *mailFrom == "" || *msgID == "" || *scanFile == "" {
		logWithTimestamp("Varist-RC=1 Rating=UNKNOWN Result=MissingParameters Detections=")
		return
	}

	cfg, err := ini.Load(*configPath)
	if err != nil {
		logWithTimestamp("Varist-RC=1 Rating=UNKNOWN Result=ConfigLoadError Detections=")
		return
	}

	haHost := cfg.Section("chaser").Key("CHASER_HOST").String()
	haPort := cfg.Section("chaser").Key("CHASER_PORT").String()

	url := fmt.Sprintf("http://%s:%s/analyze?file=%s", haHost, haPort, *scanFile)

	client := &http.Client{Timeout: time.Duration(*timeoutSeconds) * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		logWithTimestamp(fmt.Sprintf("Varist-RC=1 Rating=UNKNOWN Result=HTTPRequestError:%s Detections=", sanitize(err.Error())))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logWithTimestamp(fmt.Sprintf("Varist-RC=1 Rating=UNKNOWN Result=HTTPStatus:%s Detections=", resp.Status))
		return
	}

	logFile, err := os.OpenFile(*responseFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logWithTimestamp("Varist-RC=1 Rating=UNKNOWN Result=LogOpenError Detections=")
		return
	}
	defer logFile.Close()

	_, err = logFile.WriteString(fmt.Sprintf("msgid=%s:\n", *msgID))
	if err != nil {
		logWithTimestamp("Varist-RC=1 Rating=UNKNOWN Result=LogWriteError Detections=")
		return
	}

	respBuf, err := io.ReadAll(resp.Body)
	if err != nil {
		logWithTimestamp("Varist-RC=1 Rating=UNKNOWN Result=ReadError Detections=")
		return
	}

	_, err = logFile.Write(respBuf)
	if err != nil {
		logWithTimestamp("Varist-RC=1 Rating=UNKNOWN Result=LogWriteError Detections=")
		return
	}
	_, _ = logFile.WriteString("")

	p := parser.NewProcessor()
	err = p.ProcessResponse(strings.NewReader(string(respBuf)), *msgID, *rating, *verbose)
	if err == nil && *verbose {
		p.PrintCategoryCount()
	}

	if err != nil {
		logWithTimestamp(fmt.Sprintf("Varist-RC=1 Rating=UNKNOWN Result=ProcessingError:%s Detections=", sanitize(err.Error())))
		return
	}
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "", " "), "", " ")
}

func logWithTimestamp(msg string) {
	fmt.Printf("%s %s", time.Now().Format("2006-01-02T15:04:05Z07:00"), msg)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return fallback
}
