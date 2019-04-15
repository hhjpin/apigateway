package options

type Config struct {
	Server struct {
		Name             string   `json:"name"`
		ListenHost       string   `json:"listen_host"`
		ListenPort       int      `json:"listen_port"`
		ListenDomainName []string `json:"listen_domain_name"`

		Concurrency        int  `json:"concurrency"`
		DisabledKeepAlive  bool `json:"disabled_keep_alive"`
		ReadBufferSize     int  `json:"read_buffer_size"`
		WriteBufferSize    int  `json:"write_buffer_size"`
		MaxRequestBodySize int  `json:"max_request_body_size"`
		ReduceMemoryUsage  bool `json:"reduce_memory_usage"`
	} `json:"server"`

	Client struct {
		Name                string `json:"name"`
		MaxConnsPerHost     int    `json:"max_conns_per_host"`
		MaxIdleConnDuration int    `json:"max_idle_conn_duration"`
	} `json:"client"`

	Etcd struct {
		Name      string   `json:"name"`
		Endpoints []string `json:"endpoints"`
		Username  string   `json:"username"`
		Password  string   `json:"password"`

		AutoSyncInterval     int `json:"auto_sync_interval"`
		DialTimeout          int `json:"dial_timeout"`
		DialKeepAliveTime    int `json:"dial_keep_alive_time"`
		DialKeepAliveTimeout int `json:"dial_keep_alive_timeout"`
	} `json:"etcd"`

	Limiter struct {
		DefaultLimit            int `json:"default_limit"`
		DefaultConsumePerPeriod int `json:"default_consume_per_period"`
		DefaultConsumePeriod    int `json:"default_consume_period"`
		LimiterChanLength       int `json:"limiter_chan_length"`

		DefaultBlackList []struct {
			IP        string `json:"ip"`
			Limit     int    `json:"limit"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"default_black_list"`
	} `json:"limiter"`

	Counter struct {
		ShardNumber       int `json:"shard_number"`
		PersistencePeriod int `json:"persistence_period"`
	} `json:"counter"`

	Auth struct {
		Redis struct {
			Addr     string `json:"addr"`
			Password string `json:"password"`
			Database string `json:"database"`
		} `json:"redis"`
	} `json:"auth"`
}
