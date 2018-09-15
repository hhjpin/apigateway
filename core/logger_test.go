package core

import (
	"github.com/fatih/color"
	"log"
	"testing"
)

func TestLogger(t *testing.T) {
	a := color.HiGreenString("[INFO]")
	log.Print(a)
}
