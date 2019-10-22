package golang

const (
	RouterDefinitionPrefix  = "/Router/"
	ServiceDefinitionPrefix = "/Service/"

	NodeDefinition        = "/Node/Node-%s/"
	ServiceDefinition     = "/Service/Service-%s/"
	HealthCheckDefinition = "/HealthCheck/HC-%s/"
	RouterDefinition      = "/Router/Router-%s/"

	IDKey          = "ID"
	NameKey        = "Name"
	HostKey        = "Host"
	NodeKey        = "Node"
	PortKey        = "Port"
	StatusKey      = "Status"
	HealthCheckKey = "HealthCheck"
	PathKey        = "Path"
	TimeoutKey     = "Timeout"
	IntervalKey    = "Interval"
	RetryKey       = "Retry"
	RetryTimeKey   = "RetryTime"
	FrontendKey    = "FrontendApi"
	BackendKey     = "BackendApi"
	ServiceKey     = "Service"
)

var (
	SlashBytes             = []byte("/")
	RouterDefinitionBytes  = []byte("/Router/")
	RouterPrefixBytes      = []byte("Router-")
	ServiceDefinitionBytes = []byte("/Service/")
	ServiceKeyBytes        = []byte("Service")
	NodeKeyBytes           = []byte("Node")
)
