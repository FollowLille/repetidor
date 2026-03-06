package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAppEnv     = "dev"
	defaultHTTPHost   = "127.0.0.1"
	defaultHTTPPort   = 8080
	defaultSQLitePath = "./repetidor.sqlite3"
	defaultLogLevel   = "debug"
	defaultLogFormat  = "text"

	envAppEnv     = "REPETIDOR_APP_ENV"
	envHTTPHost   = "REPETIDOR_HTTP_HOST"
	envHTTPPort   = "REPETIDOR_HTTP_PORT"
	envSQLitePath = "REPETIDOR_SQLITE_PATH"
	envLogLevel   = "REPETIDOR_LOG_LEVEL"
	envLogFormat  = "REPETIDOR_LOG_FORMAT"
)

// Config stores runtime configuration for the web application.
type Config struct {
	AppEnv     string
	HTTPHost   string
	HTTPPort   int
	SQLitePath string
	LogLevel   string
	LogFormat  string
}

// Load builds application config using defaults, env variables or flags
// priority: flags > env > defaults
func Load() (Config, error) {
	cfg := defaultConfig()

	applyEnv(&cfg)

	cfg = applyFlags(cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		AppEnv:     defaultAppEnv,
		HTTPHost:   defaultHTTPHost,
		HTTPPort:   defaultHTTPPort,
		SQLitePath: defaultSQLitePath,
		LogLevel:   defaultLogLevel,
		LogFormat:  defaultLogFormat,
	}
}

func applyEnv(cfg *Config) {
	if value, ok := os.LookupEnv(envAppEnv); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			log.Printf("config warning: %s is empty, using previous value %q", envAppEnv, cfg.AppEnv)
		} else {
			cfg.AppEnv = value
		}
	}

	if value, ok := os.LookupEnv(envHTTPHost); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			log.Printf("config warning: %s is empty, using previous value %q", envHTTPHost, cfg.HTTPHost)
		} else {
			cfg.HTTPHost = value
		}
	}

	if value, ok := os.LookupEnv(envHTTPPort); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			log.Printf("config warning: %s is empty, using previous value %d", envHTTPPort, cfg.HTTPPort)
		} else {
			port, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("config warning: invalid %s value %q, using previous value %d", envHTTPPort, value, cfg.HTTPPort)
			} else {
				cfg.HTTPPort = port
			}
		}
	}

	if value, ok := os.LookupEnv(envSQLitePath); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			log.Printf("config warning: %s is empty, using previous value %q", envSQLitePath, cfg.SQLitePath)
		} else {
			cfg.SQLitePath = value
		}
	}

	if value, ok := os.LookupEnv(envLogLevel); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			log.Printf("config warning: %s is empty, using previous value %q", envLogLevel, cfg.LogLevel)
		} else {
			cfg.LogLevel = strings.ToLower(value)
		}
	}

	if value, ok := os.LookupEnv(envLogFormat); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			log.Printf("config warning: %s is empty, using previous value %q", envLogFormat, cfg.LogFormat)
		} else {
			cfg.LogFormat = strings.ToLower(value)
		}
	}
}

func applyFlags(base Config) Config {
	env := flag.String("env", base.AppEnv, "application environment (dev, prod)")
	host := flag.String("host", base.HTTPHost, "HTTP host")
	port := flag.Int("port", base.HTTPPort, "HTTP port")
	sqlitePath := flag.String("sqlite-path", base.SQLitePath, "SQLite database path")
	logLevel := flag.String("log-level", base.LogLevel, "log level (debug, info, warn, error)")
	logFormat := flag.String("log-format", base.LogFormat, "log format (text, json)")

	flag.Parse()

	return Config{
		AppEnv:     strings.TrimSpace(*env),
		HTTPHost:   strings.TrimSpace(*host),
		HTTPPort:   *port,
		SQLitePath: strings.TrimSpace(*sqlitePath),
		LogLevel:   strings.ToLower(strings.TrimSpace(*logLevel)),
		LogFormat:  strings.ToLower(strings.TrimSpace(*logFormat)),
	}
}

func validate(cfg Config) error {
	var errs []string

	switch cfg.AppEnv {
	case "dev", "prod":
	default:
		errs = append(errs, fmt.Sprintf("AppEnv error: %q is not supported, allowed values are dev or prod", cfg.AppEnv))
	}

	if cfg.HTTPHost == "" {
		errs = append(errs, "HTTPHost error: host is required")
	}

	if cfg.HTTPPort <= 0 || cfg.HTTPPort > 65535 {
		errs = append(errs, "HTTPPort error: http port must be between 1 and 65535")
	}

	if cfg.SQLitePath == "" {
		errs = append(errs, "SQLitePath error: path must not be empty")
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Sprintf("LogLevel error: %q is not supported, allowed values are debug, info, warn, error", cfg.LogLevel))
	}

	switch cfg.LogFormat {
	case "text", "json":
	default:
		errs = append(errs, fmt.Sprintf("LogFormat error: %q is not supported, allowed values are text or json", cfg.LogFormat))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

// Address returns the HTTP bind address in host:port form.
func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}
