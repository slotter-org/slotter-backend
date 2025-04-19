package utils

import (
  "os"
  "strconv"

  "github.com/yungbote/slotter/backend/internal/logger"
)

func GetEnv(key, defaultVal string, log *logger.Logger) string {
  if log != nil {
    log = log.With("env_var", key)
    log.Debug("Attempting to load environment variable (string)...")
  }
  val, ok := os.LookupEnv(key)
  if !ok {
    if log != nil {
      log.Debug("Environment variable not found, using default value", "defaultValue", defaultVal)
    }
    return defaultVal
  }
  if log != nil {
    log.Debug("Environment variable found (string), using environment variable value", "value", val)
  }
  return val
}

func GetEnvAsInt(key string, defaultVal int, log *logger.Logger) int {
  if log != nil {
    log = log.With("env_var", key)
    log.Debug("Attempting to load environment variable (int)...")
  }
  valStr, ok := os.LookupEnv(key)
  if !ok {
    if log != nil {
      log.Debug("Environment variable not found, using default int", "defaultVal", defaultVal)
    }
    return defaultVal
  }
  i, err := strconv.Atoi(valStr)
  if err != nil {
    if log != nil {
      log.Debug("Environment variable could not be parsed as int, using default", "providedVal", valStr, "defaultVal", defaultVal, "error", err)
    }
    return defaultVal
  }
  if log != nil {
    log.Debug("Environment variable found (int), using environment variable value", "value", i)
  }
  return i
}

