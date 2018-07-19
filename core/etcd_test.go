package core

import (
	"testing"
	"context"
	"log"
)

func TestConnectToEtcd(t *testing.T) {
	client := ConnectToEtcd()
	res, _ := client.Get(context.TODO(), "testkey1")
	log.Print(res)
}
