package utils

import (
	"log"
	"os"
)

var Logger = log.New(os.Stdout, "[SERVER] ", log.LstdFlags)
