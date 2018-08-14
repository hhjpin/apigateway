package utils

import (
	"strings"
	"strconv"
	"log"
	"net"
)

func GetHardwareAddressAsLong() []int64 {
	var long []int64

	inter, _ := net.Interfaces()
	for _, i:= range inter {
		macStr := i.HardwareAddr.String()
		if macStr != "" {
			hexStr := strings.Join(strings.Split(macStr, ":"), "")

			integer, err := strconv.ParseInt(hexStr, 16, 64)
			if err != nil {
				log.Print(err)
			} else {
				long = append(long, integer)
			}
		}
	}
	return long
}
