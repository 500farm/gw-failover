package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

var configFile = kingpin.Flag(
	"config-file",
	"Path to config file (if the file does not exist, it will created with default values).",
).String()

type ConfigType struct {
	InactiveRouteMetric int           `yaml:"inactive_route_metric"`
	DeactivateThreshold time.Duration `yaml:"deactivate_threshold"`
	ActivateThreshold   time.Duration `yaml:"activate_threshold"`
	PingInterval        time.Duration `yaml:"ping_interval"`
	ReplyTimeout        time.Duration `yaml:"reply_timeout"`
	DryRun              bool          `yaml:"dry_run"`
	Ipv6                bool          `yaml:"ipv6"`
}

var Config = ConfigType{
	InactiveRouteMetric: 10000,
	DeactivateThreshold: 30 * time.Second,
	ActivateThreshold:   120 * time.Second,
	PingInterval:        time.Second,
	ReplyTimeout:        5 * time.Second,
	DryRun:              false,
	Ipv6:                false,
}

func (c *ConfigType) Load() error {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *configFile != "" {
		t, err := ioutil.ReadFile(*configFile)
		if err != nil {
			if os.IsNotExist(err) {
				return c.Write(*configFile)
			}
			return err
		}
		err = yaml.Unmarshal(t, c)
		if err != nil {
			return err
		}
	}

	log.Infof("Config: " + c.String())
	return nil
}

func (c *ConfigType) Write(file string) error {
	y, _ := yaml.Marshal(*c)
	err := ioutil.WriteFile(file, y, 0644)
	if err != nil {
		return err
	}
	log.Infof("Written config file %s", file)
	log.Infof("Config: " + c.String())
	return nil
}

func (c *ConfigType) String() string {
	j, _ := json.MarshalIndent(*c, "", "    ")
	return string(j)
}
