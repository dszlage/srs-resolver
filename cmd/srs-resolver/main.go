/*
srs-resolver - SRS decoder for autoresponders
Copyright (C) 2025 Damian Szlage / Umbrella Dev Systems / DriftZone.pl
https://github.com/dszlage/srs-resolver

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	logToStdOut     = flag.Bool("logtostdout", false, "Send log to stdout")
	showVersion     = flag.Bool("v", false, "Show info")
	showVersionLong = flag.Bool("version", false, "Show info")
)

// LogLevel type and constants
type LogLevel int

const version = "1.0.0"
const notAllowedChars = " <>(),;=\"" // Characters not allowed in a clean email address
const (
	LogError LogLevel = iota
	LogInfo
	LogDebug
)

var currentLogLevel = LogError

// Config struct for TOML parsing
type Config struct {
	Listen          string `toml:"listen"`
	LogFile         string `toml:"log_file"`
	LogLevel        string `toml:"log_level"`
	FallbackAddress string `toml:"fallback_address"`
}

// LoadConfig loads TOML config from file
func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// InitLogging sets log output and level
func InitLogging(cfg *Config) error {
	// Set log level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		currentLogLevel = LogDebug
	case "info":
		currentLogLevel = LogInfo
	case "error":
		currentLogLevel = LogError
	default:
		currentLogLevel = LogError
	}

	// Set log file output
	if cfg.LogFile != "" && !*logToStdOut {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		log.SetOutput(f)
	} else if *logToStdOut {
		log.SetOutput(os.Stdout)
	}

	log.SetFlags(log.LstdFlags)
	return nil
}

// Logging helpers
func logError(format string, a ...any) {
	log.Printf("[ERROR] "+format, a...)
}

func logInfo(format string, a ...any) {
	if currentLogLevel >= LogInfo {
		log.Printf("[INFO] "+format, a...)
	}
}

func logDebug(format string, a ...any) {
	if currentLogLevel >= LogDebug {
		log.Printf("[DEBUG] "+format, a...)
	}
}

func main() {
	flag.Parse()
	if *showVersion || *showVersionLong {
		fmt.Println("srs-resolver - Lightweight SRS decoder for Postfix autoresponders\n",
			"Version: ", version, "\n",
			"Copyright © 2025 Damian Szlage / Umbrella Dev Systems / DriftZone.pl\n",
			"\nLicense: GNU General Public License v3 or later")
	}

	cfg, err := LoadConfig("/etc/srs-resolver/srs-resolver.conf")

	if err != nil {
		fmt.Println("Config error: ", err)
		os.Exit(1)
	}

	if err := InitLogging(cfg); err != nil {
		fmt.Println("Logging error: ", err)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Fatalf("Błąd podczas nasłuchu: %v", err)
	}
	logInfo("Nasłuch na %s", cfg.Listen)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Błąd połączenia: %v", err)
			continue
		}
		go handle(conn, cfg)
	}
}

func handle(conn net.Conn, cfg *Config) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(conn, "500 read error\n")
		return
	}

	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "get ") {
		fmt.Fprintf(conn, "500 invalid request\n")
		return
	}

	address := strings.TrimSpace(line[4:])

	// Szybka walidacja - jeśli to nie SRS, sprawdź czy to poprawny email
	if !strings.HasPrefix(address, "SRS0=") && !strings.HasPrefix(address, "SRS1=") {
		if isCleanEmail(address) {
			logDebug("Poprawny adres: %s, nie wymaga rozwiązywania", address)
			fmt.Fprintf(conn, "200 %s\n", address)
			return
		}
		// Nie jest SRS ani poprawnym emailem
		if cfg.FallbackAddress != "" {
			logError("Błędny adres: %s, zwracam fallback_address: %s", address, cfg.FallbackAddress)
			fmt.Fprintf(conn, "200 %s\n", cfg.FallbackAddress)
		} else {
			logError("Błędny adres: %s, brak ustawionego fallback_address, 500 invalid request", address)
			fmt.Fprintf(conn, "500 invalid request\n")
		}
		return
	}

	// To jest adres SRS - dekoduj go
	decoded, err := decodeSRS(address)
	if err != nil {
		if cfg.FallbackAddress != "" {
			logError("Błędny SRS: %s, zwracam fallback_address: %s", address, cfg.FallbackAddress)
			fmt.Fprintf(conn, "200 %s\n", cfg.FallbackAddress)
		} else {
			logError("Błędny SRS: %s, brak ustawionego fallback_address, 500 invalid request", address)
			fmt.Fprintf(conn, "500 invalid request\n")
		}
	} else {
		logInfo("Rozwiązano: %s → %s", address, decoded)
		fmt.Fprintf(conn, "200 %s\n", decoded)
	}
}

func decodeSRS(srs string) (string, error) {
	// Prefiks SRS sprawdzany jest już w funkcji handle(), nie trzeba powtarzać walidacji

	// Używamy wyrażenia regularnego, które przechwytuje *wszystko* po 4. znaku '='
	// SRS0=hash=time=domain=full_local_part@something
	// SRS1=hash=time=domain=full_local_part@something
	parts := strings.SplitN(srs, "=", 5)
	if len(parts) != 5 {
		return "", fmt.Errorf("nieprawidłowy format SRS - nieprawidłowa liczba części")
	}

	// parts[3] = domena oryginalna
	// parts[4] = lokalna część (która może zawierać @forwarder)
	domain := parts[3]
	local := parts[4]

	// Jeśli wygląda już jak pełny adres, to tylko przestawiamy domenę
	if strings.Contains(local, "@") {
		// np. damian.szlage@attmail.pl → damian.szlage@driftzone24.pl
		user := strings.Split(local, "@")[0]
		return fmt.Sprintf("%s@%s", user, domain), nil
	}

	// Zwykły przypadek
	return fmt.Sprintf("%s@%s", local, domain), nil
}

func isCleanEmail(s string) bool {
	// Nie może zawierać spacji, nawiasów, przecinków itd.
	if strings.ContainsAny(s, notAllowedChars) {
		return false
	}

	// Musi zawierać dokładnie jedno "@"
	if strings.Count(s, "@") != 1 {
		return false
	}
	// Podział na lokalną część i domenę
	parts := strings.Split(s, "@")
	if len(parts[0]) == 0 || len(parts[1]) < 3 {
		return false
	}
	// Domena powinna zawierać przynajmniej jedną kropkę
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}
