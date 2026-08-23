package config

import (
	"os"
	"strconv"
)

func Port() int {
	p, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil || p < 1 || p > 65535 {
		return 8080
	}
	return p
}

func MaxHistory() int {
	p, err := strconv.Atoi(os.Getenv("MAX_HISTORY"))
	if err != nil || p < 10 {
		return 500
	}
	return p
}

func MaxInspectionWorkers() int {
	p, err := strconv.Atoi(os.Getenv("MAX_INSPECTION_WORKERS"))
	if err != nil || p < 1 {
		return 2
	}
	return p
}
