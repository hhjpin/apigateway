package conf

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strings"
)

const (
	defaultLogPath = "./conf.yaml"
)

type Config struct {
	Server struct {
		Name             string   `yaml:"Name"`
		ListenHost       string   `yaml:"ListenHost"`
		ListenPort       int      `yaml:"ListenPort"`
		ListenDomainName []string `yaml:"ListenDomainName"`

		Concurrency        int  `yaml:"Concurrency"`
		DisabledKeepAlive  bool `yaml:"DisabledKeepAlive"`
		ReadBufferSize     int  `yaml:"ReadBufferSize"`
		WriteBufferSize    int  `yaml:"WriteBufferSize"`
		MaxRequestBodySize int  `yaml:"MaxRequestBodySize"`
		ReduceMemoryUsage  bool `yaml:"ReduceMemoryUsage"`
	} `yaml:"Server"`

	Client struct {
		Name                string `yaml:"Name"`
		MaxConnsPerHost     int    `yaml:"MaxConnsPerHost"`
		MaxIdleConnDuration int    `yaml:"MaxIdleConnDuration"`
	} `yaml:"client"`

	Etcd struct {
		Name      string   `yaml:"Name"`
		Endpoints []string `yaml:"Endpoints"`
		Username  string   `yaml:"Username"`
		Password  string   `yaml:"Password"`

		AutoSyncInterval     int `yaml:"AutoSyncInterval"`
		DialTimeout          int `yaml:"DialTimeout"`
		DialKeepAliveTime    int `yaml:"DialKeepAliveTime"`
		DialKeepAliveTimeout int `yaml:"DialKeepAliveTimeout"`
	} `yaml:"Etcd"`

	Middleware struct {
		Limiter struct {
			DefaultLimit            uint64 `yaml:"DefaultLimit"`
			DefaultConsumePerPeriod uint64 `yaml:"DefaultConsumeNumberPerPeriod"`
			DefaultConsumePeriod    int64  `yaml:"DefaultConsumePeriod"`
			LimiterChanLength       uint64 `yaml:"LimiterChanLength"`

			DefaultBlackList []struct {
				IP        string `yaml:"IP"`
				Limit     uint64 `yaml:"Limit"`
				ExpiresAt int64  `yaml:"ExpiresAt"`
			} `yaml:"DefaultBlackList"`
		} `yaml:"Limiter"`

		Counter struct {
			ShardNumber       int `yaml:"ShardNumber"`
			PersistencePeriod int `yaml:"PersistencePeriod"`
		} `yaml:"Counter"`

		Auth struct {
			Redis struct {
				Addr     string `yaml:"Addr"`
				Password string `yaml:"Password"`
				Database string `yaml:"Database"`
			} `yaml:"Redis"`
		} `yaml:"Auth"`
	} `yaml:"Middleware"`

	DashBoard struct {
		Enable       bool   `yaml:"Enable"`
		ListenHost   string `yaml:"ListenHost"`
		ListenPort   int    `yaml:"ListenPort"`
		RoutePrefix  string `yaml:"RoutePrefix"`
		RequestModel string `yaml:"RequestModel"`
		Token        string `yaml:"Token"`
	} `yaml:"DashBoard"`
}

var (
	Conf *Config
)

func ReadConfig(path ...string) *Config {
	var f *os.File
	var err error
	var config Config
	var isProductionEnv bool
	var configFilePath string

	// env IS_PRODUCTION
	if os.Getenv("IS_PRODUCTION") == "1" {
		isProductionEnv = true
	} else {
		isProductionEnv = false
	}

	// env CONFIG_PATH
	if len(path) == 1 {
		configFilePath = path[0]
	} else if len(path) == 0 {
		configFilePath = os.Getenv("CONFIG_PATH")
		if configFilePath == "" {
			configFilePath = defaultLogPath
		}
	} else {
		log.Fatal("only one path could be passed in")
	}

	if !isProductionEnv {
		fp := strings.Split(configFilePath, ".")
		fpLen := len(fp)
		if fpLen > 1 {
			fp[fpLen-2] += "_test"
			configFilePath = strings.Join(fp, ".")
		} else if fpLen == 1 {
			configFilePath += "_test"
		} else {
			log.Fatal(fmt.Sprintf("lack of config file path, %s", configFilePath))
		}
	}
	f, err = os.OpenFile(configFilePath, os.O_RDONLY, 0666)
	defer func(fd *os.File) {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}(f)
	if err != nil {
		log.Fatal(err)
	}

	length, _ := f.Seek(0, 2)
	conf := make([]byte, length)
	if _, err := f.Seek(0, 0); err != nil {
		panic(err)
	}
	_, err = f.Read(conf)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(conf, &config)
	if err != nil {
		log.Fatal(err)
	}
	return &config
}

func init() {
	Conf = ReadConfig()
}
