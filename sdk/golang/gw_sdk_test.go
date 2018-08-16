package golang

import (
	"log"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestApiGatewayRegistrant_Register(t *testing.T) {
	inter, _ := net.Interfaces()
	for _, i := range inter {
		macStr := i.HardwareAddr.String()
		if macStr != "" {
			hexStr := strings.Join(strings.Split(macStr, ":"), "")

			intStr, err := strconv.ParseInt(hexStr, 16, 64)
			if err != nil {
				log.Print(err)
			}
			log.Print(intStr)
		}
	}
}

func ExampleApiGatewayRegistrant_Register() {

}
