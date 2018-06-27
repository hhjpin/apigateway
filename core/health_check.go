package core

type HealthCheck struct {
	path []byte

	// timeout default is 5 sec. When timed out, will retry to check again based on 'retry' switch on or not
	timeout   uint8
	interval  uint8
	retry     bool
	retryTime uint8
}

func CreateHealthCheck(path []byte, timeout uint8, interval uint8, retry bool, retryTime uint8) *HealthCheck {

	return &HealthCheck{
		path:      path,
		timeout:   timeout,
		interval:  interval,
		retry:     retry,
		retryTime: retryTime,
	}
}
