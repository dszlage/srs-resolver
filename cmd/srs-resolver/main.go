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

	// For dropping privileges
	"os/user"
	"strconv"
	"syscall"

	// For TOML parsing
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
	DropUser        string `toml:"drop_user"`
	DropGroup       string `toml:"drop_group"`
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

// / Logging helpers
func logFatal(format string, a ...any) {
	log.Fatalf("[FATAL] "+format, a...)
}
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
		fmt.Println("[FATAL] Config error: ", err)
		os.Exit(1)
	}

	if err := InitLogging(cfg); err != nil {
		fmt.Println("[FATAL] Logging error: ", err)
		os.Exit(1)
	}

	logInfo("srs-resolver version %s starting...", version)

	// Drop privileges if configured
	if err := dropPrivileges(cfg.DropUser, cfg.DropGroup); err != nil {
		logFatal("Dropping privileges failed: %v", err)
	}

	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		logFatal("Listen error: %v", err)
	}
	logInfo("Listening on %s", cfg.Listen)

	for {
		conn, err := ln.Accept()
		if err != nil {
			logError("Connection error: %v", err)
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

	// Fast validation - if not SRS, check if it's a clean email
	// We skip full validation for best performance
	if !strings.HasPrefix(address, "SRS0=") && !strings.HasPrefix(address, "SRS1=") {
		if isCleanEmail(address) {
			logDebug("Address: %s, no decoding required", address)
			fmt.Fprintf(conn, "200 %s\n", address)
			return
		}
		// Fallback or error
		if cfg.FallbackAddress != "" {
			logError("Invalid address: %s, returning fallback_address: %s", address, cfg.FallbackAddress)
			fmt.Fprintf(conn, "200 %s\n", cfg.FallbackAddress)
		} else {
			logError("Invalid address: %s, no fallback_address set, 500 invalid request", address)
			fmt.Fprintf(conn, "500 invalid request\n")
		}
		return
	}

	// It's SRS, try to decode
	decoded, err := decodeSRS(address)
	if err != nil {
		if cfg.FallbackAddress != "" {
			logError("Invalid SRS: %s (%v), returning fallback_address: %s", address, err, cfg.FallbackAddress)
			fmt.Fprintf(conn, "200 %s\n", cfg.FallbackAddress)
		} else {
			logError("Invalid SRS: %s (%v), no fallback_address set, 500 invalid request", address, err)
			fmt.Fprintf(conn, "500 invalid request\n")
		}
	} else {
		logInfo("Resolved: %s → %s", address, decoded)
		fmt.Fprintf(conn, "200 %s\n", decoded)
	}
}

func decodeSRS(srs string) (string, error) {

	// SRS0=hash=time=domain=full_local_part@something
	// SRS1=hash=time=domain=full_local_part@something
	parts := strings.SplitN(srs, "=", 5)
	if len(parts) != 5 {
		return "", fmt.Errorf("SRS format - wrong number of parts")
	}

	// parts[3] = original domain
	// parts[4] = local part (which may contain @forwarder)
	domain := parts[3]
	local := parts[4]

	// If it already looks like a full address, just switch the domain
	if strings.Contains(local, "@") {
		// e.g. user@forwarder.com → user@domain.com
		user := strings.Split(local, "@")[0]
		return fmt.Sprintf("%s@%s", user, domain), nil
	}

	// Normal case
	return fmt.Sprintf("%s@%s", local, domain), nil
}

func isCleanEmail(s string) bool {
	// RFC 5321, 5322 Not allowed characters
	if strings.ContainsAny(s, notAllowedChars) {
		return false
	}

	// Must contain exactly one "@" symbol
	if strings.Count(s, "@") != 1 {
		return false
	}
	// Split into local part and domain
	parts := strings.Split(s, "@")
	if len(parts[0]) == 0 || len(parts[1]) < 3 {
		return false
	}
	// Domain must contain at least one dot
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

func dropPrivileges(targetUser string, targetGroup string) error {

	// ========== set GID section ==========
	if targetGroup != "" {

		g, err := user.LookupGroup(targetGroup)
		if err != nil {
			return fmt.Errorf("lookup group: %v", err)
		}

		gid, err := strconv.Atoi(g.Gid)
		if err != nil {
			return fmt.Errorf("bad GID: %v", err)
		}

		if err := syscall.Setgid(gid); err != nil {
			return fmt.Errorf("setgid failed: %v", err)
		}
	}

	// ========== set UID section ==========
	if targetUser != "" {
		u, err := user.Lookup(targetUser)
		if err != nil {
			return fmt.Errorf("lookup user: %v", err)
		}

		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			return fmt.Errorf("bad UID: %v", err)
		}

		if err := syscall.Setuid(uid); err != nil {
			return fmt.Errorf("setuid failed: %v", err)
		}
	}

	// ========== check UId/GId section ==========
	newUser, err := user.LookupId(strconv.Itoa(syscall.Geteuid()))
	if err != nil {
		return fmt.Errorf("lookup new user: %v", err)
	}

	newGroup, err := user.LookupGroupId(strconv.Itoa(syscall.Getegid()))
	if err != nil {
		return fmt.Errorf("lookup new group: %v", err)
	}

	if newUser.Username == "root" {
		logError("Warning! Running as root user! Not dropping privileges! Be careful!")
	}
	if newGroup.Name == "root" {
		logError("Warning! Running as root group! Not dropping privileges! Be careful!")
	}
	logInfo("Running as user: %s (UID %s), group: %s (GID %s)", newUser.Username, newUser.Uid, newGroup.Name, newGroup.Gid)
	return nil
}
