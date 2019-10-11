package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/arcanericky/opiekey"
)

type pamTest struct {
	user string
}

func (p pamTest) GetChallengeResponse(user, prompt string) string {
	words := strings.Split(prompt, " ")
	seq, _ := strconv.Atoi(words[1])
	seed := words[2]
	response := opiekey.ComputeWordResponse(seq, seed, "testpassphrase", opiekey.MD5)

	return response
}

func (p pamTest) GetUser() string {
	return p.user
}

type testReader struct{}

func (tr testReader) Read(p []byte) (int, error) {
	return 0, errors.New("reader error")
}

func createTestConfig(file, configData string) {
	ioutil.WriteFile(file, []byte(configData), 0600)
}

func createTestConfigFileWithoutPassphrase(file string) {
	createTestConfig(file, `{
		"users": [
			{
				"name": "alldefaults"
			}
		]
	}`)
}

func createTestConfigFileWithoutDefaults(file string) {
	createTestConfig(file, `{
		"users": [
			{
				"name": "alldefaults",
				"passphrase": "testpassphrase"
			}
		]
	}`)
}

func createTestConfigFileWithDefaults(file string) {
	createTestConfig(file, `{
		"defaults":
		{
			"maxseq": 99,
			"passphrase": "defaultpassphrase",
			"retries": 3,
			"seedlen": 6
		},
		"users": [
			{
				"name": "alldefaults"
			},
			{
				"name": "allset",
				"maxseq": 7331,
				"passphrase": "testpassphrase",
				"retries": 42,
				"seedlen": 9
			}
		]
	}`)
}

func TestGetRandomSequence(t *testing.T) {
	const max = 5

	for i := 0; i < max*2; i++ {
		x := getRandomSequence(max)

		if x < 1 || x > max {
			t.Error("Generated random sequence out of range")
		}
	}
}

func TestGetRandomSeed(t *testing.T) {
	const max = 6

	for i := 0; i < max*3; i++ {
		x := len(getRandomSeed(max))

		if x != max {
			t.Error("Generated seed not correct length")
		}
	}
}

func TestGetUserConfig(t *testing.T) {
	opieConfig := "opie.json"

	// config file does not exist
	expectedUser := ""
	config := getUserConfig("alldefaults", opieConfig)

	if config.name != expectedUser {
		t.Errorf("Config file does not exist but user was returned")
	}

	// config file contains invalid json
	ioutil.WriteFile(opieConfig, []byte("invalidjson"), 0600)
	expectedUser = ""
	config = getUserConfig("alldefaults", opieConfig)
	os.Remove(opieConfig)

	if config.name != expectedUser {
		t.Errorf("Config file contains invalid json but user was returned")
	}

	// config file has invalid permissions
	ioutil.WriteFile(opieConfig, []byte("{}"), 0604)
	expectedUser = ""
	config = getUserConfig("alldefaults", opieConfig)
	os.Remove(opieConfig)

	if config.name != expectedUser {
		t.Errorf("Config file has invalid permissions but user was returned")
	}

	// config file could not be read
	expectedUser = ""
	config = getUserConfigFromReader("alldefaults", testReader{})

	if config.name != expectedUser {
		t.Errorf("Config file cannot be read but user was returned")
	}

	// test hardcoded defaults
	createTestConfigFileWithoutDefaults(opieConfig)
	expectedUser = "alldefaults"
	config = getUserConfig("alldefaults", opieConfig)
	if config.name != expectedUser {
		t.Errorf("User get not successful. Expected: %s. Actual: %s", expectedUser, config.name)
	}

	if config.maxSeq != defaultMaxSeq {
		t.Errorf("MaxSeq not properly defaulted. Expected: %d. Actual: %d", defaultMaxSeq, config.maxSeq)
	}

	if config.retries != defaultRetries {
		t.Errorf("Retries not properly defaulted. Expected: %d. Actual: %d", defaultRetries, config.retries)
	}

	if config.seedLen != defaultSeedLen {
		t.Errorf("SeedLen not properly defaulted. Expected: %d. Actual: %d", defaultSeedLen, config.seedLen)
	}

	// No user returned if passphrase not configured
	createTestConfigFileWithoutPassphrase(opieConfig)
	expectedUser = ""
	config = getUserConfig("alldefaults", opieConfig)
	if config.name != expectedUser {
		t.Errorf("Passphrase not configured but user was returned")
	}

	// Everything else
	createTestConfigFileWithDefaults(opieConfig)
	defer os.Remove(opieConfig)

	for _, test := range []struct {
		expectedUser       string
		expectedMaxSeq     int
		expectedPassphrase string
		expectedRetries    int
		expectedSeedLen    int
	}{
		{"alldefaults", 99, "defaultpassphrase", 3, 6},
		{"allset", 7331, "testpassphrase", 42, 9},
	} {
		config := getUserConfig(test.expectedUser, opieConfig)

		if config.name != test.expectedUser {
			t.Errorf("User get not successful. Expected: %s. Actual: %s", test.expectedUser, config.name)
		}

		if config.maxSeq != test.expectedMaxSeq {
			t.Errorf("MaxSeq not properly populated. Expected: %d. Actual: %d", test.expectedMaxSeq, config.maxSeq)
		}

		if config.passphrase != test.expectedPassphrase {
			t.Errorf("Passphrase not properly populated. Expected: %s. Actual: %s", test.expectedPassphrase, config.passphrase)
		}

		if config.retries != test.expectedRetries {
			t.Errorf("Retries not properly populated. Expected: %d. Actual: %d", test.expectedRetries, config.retries)
		}

		if config.seedLen != test.expectedSeedLen {
			t.Errorf("SeedLen not properly populated. Expected: %d. Actual: %d", test.expectedSeedLen, config.seedLen)
		}
	}
}

func TestGetOPIEConfig(t *testing.T) {
	expectedValue := "configvalue"
	parameter := fmt.Sprintf("config=%s", expectedValue)
	actualValue := getOPIEConfigItem("config", []string{parameter})

	if actualValue != expectedValue {
		t.Errorf("Config item not retrieved. Expected: %s. Actual: %s", expectedValue, actualValue)
	}

	expectedValue = ""
	actualValue = getOPIEConfigItem("noentry", []string{parameter})
	if actualValue != expectedValue {
		t.Errorf("Config item should not be found. Expected: %s. Actual: %s", expectedValue, actualValue)
	}
}

func TestAuthenticate(t *testing.T) {
	opieConfig := "opie.json"
	opieParameters := []string{fmt.Sprintf("config=%s", opieConfig)}

	createTestConfigFileWithDefaults(opieConfig)
	defer os.Remove(opieConfig)
	authenticate(pamTest{"allset"}, 0, opieParameters)

	// config user not found
	authenticate(pamTest{"nosuchuser"}, 0, opieParameters)

	// no config file specified
	authenticate(pamTest{"allset"}, 0, []string{})

}
