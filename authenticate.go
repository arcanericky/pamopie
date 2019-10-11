package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/syslog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/arcanericky/opiekey"
)

const (
	defaultMaxSeq  = 499
	defaultRetries = 1
	defaultSeedLen = 6

	pamSuccess     = 0
	pamAuthErr     = 1
	pamCredUnavail = 2
)

type userConfig struct {
	name       string
	maxSeq     int
	passphrase string
	retries    int
	seedLen    int
}

type pamFuncs interface {
	GetChallengeResponse(string, string) string
	GetUser() string
}

func opieSyslog(data string) {
	if lw, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_AUTHPRIV, "opie"); err == nil {
		fmt.Fprintf(lw, data)
		lw.Close()
	}
}

func getRandomSequence(max int) int {
	return rand.Intn(max) + 1
}

func getRandomSeed(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		j := rand.Intn(36)
		if j >= 0 && j <= 9 {
			bytes[i] = byte(j + 48)
		} else {
			bytes[i] = byte(j - 10 + 97)
		}
	}

	return string(bytes)
}

func getUserConfigFromReader(userName string, configReader io.Reader) userConfig {
	var config userConfig

	type DefaultValues struct {
		MaxSeq     int    `json:"maxseq"`
		Passphrase string `json:"passphrase"`
		Retries    int    `json:"retries"`
		SeedLen    int    `json:"seedlen"`
	}

	type User struct {
		Name       string `json:"name"`
		MaxSeq     int    `json:"maxseq"`
		Passphrase string `json:"passphrase"`
		Retries    int    `json:"retries"`
		SeedLen    int    `json:"seedlen"`
	}

	type ConfigData struct {
		Defaults DefaultValues `json:"defaults"`
		Users    []User        `json:"users"`
	}

	var configData ConfigData

	if byteValue, err := ioutil.ReadAll(configReader); err == nil {
		if json.Unmarshal(byteValue, &configData) == nil {
			for _, user := range configData.Users {
				if userName == user.Name {
					config.name = user.Name

					if user.MaxSeq > 0 {
						config.maxSeq = user.MaxSeq
					} else if configData.Defaults.MaxSeq > 0 {
						config.maxSeq = configData.Defaults.MaxSeq
					} else {
						config.maxSeq = defaultMaxSeq
					}

					if len(user.Passphrase) > 0 {
						config.passphrase = user.Passphrase
					} else if len(configData.Defaults.Passphrase) > 0 {
						config.passphrase = configData.Defaults.Passphrase
					} else {
						opieSyslog("No passphrase configured")
						config.name = ""
					}

					if user.Retries > 0 {
						config.retries = user.Retries
					} else if configData.Defaults.Retries > 0 {
						config.retries = configData.Defaults.Retries
					} else {
						config.retries = defaultRetries
					}

					if user.SeedLen > 0 {
						config.seedLen = user.SeedLen
					} else if configData.Defaults.SeedLen > 0 {
						config.seedLen = configData.Defaults.SeedLen
					} else {
						config.seedLen = defaultSeedLen
					}
				}
			}
		} else {
			opieSyslog("Config file could not be parsed")
		}
	} else {
		opieSyslog("Config file could not be read")
	}

	return config
}

func getUserConfig(userName, configFile string) userConfig {
	var config userConfig

	info, errStat := os.Stat(configFile)
	if errStat == nil && info.Mode().Perm() != 0600 {
		opieSyslog(fmt.Sprintf("Attributes too permissible for config file %s", configFile))
	} else {
		reader, errOpen := os.Open(configFile)
		if errOpen == nil {
			defer reader.Close()
		}

		if errStat != nil || errOpen != nil {
			opieSyslog(fmt.Sprintf("Could not access config file %s", configFile))
		} else {
			config = getUserConfigFromReader(userName, reader)
		}
	}

	return config
}

func getOPIEConfigItem(item string, args []string) string {
	var value string

	for _, arg := range args {
		fields := strings.Split(arg, "=")

		if len(fields) == 2 {
			if fields[0] == item {
				value = fields[1]
			}
		}
	}

	return value
}

func authenticate(pamService pamFuncs, flags int, args []string) int {
	retval := pamAuthErr

	if user := pamService.GetUser(); len(user) > 0 {
		const promptFormat = "otp-md5 %d %s ext\nPassword: "

		opieConfigFile := getOPIEConfigItem("config", args)

		if len(opieConfigFile) == 0 {
			opieSyslog("Config file parameter not found")
		} else {
			config := getUserConfig(user, opieConfigFile)

			if len(config.name) > 0 {
				rand.Seed(time.Now().UnixNano())
				seq := getRandomSequence(config.maxSeq)
				seed := getRandomSeed(config.seedLen)
				prompt := fmt.Sprintf(promptFormat, seq, seed)
				expected := opiekey.ComputeWordResponse(seq, seed, config.passphrase, opiekey.MD5)

				for i := 0; i < config.retries && retval != pamSuccess; i++ {
					actual := pamService.GetChallengeResponse(user, prompt)

					if actual == expected {
						retval = pamSuccess
					}
				}
			} else {
				retval = pamCredUnavail
			}
		}
	}

	return retval
}
