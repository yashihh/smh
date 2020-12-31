/*
 * AMF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

var SmfConfig Config
var UERoutingConfig RoutingConfig

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	if content, err := ioutil.ReadFile(f); err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	} else {

		SmfConfig = Config{}

		if yamlErr := yaml.Unmarshal([]byte(content), &SmfConfig); yamlErr != nil {
			return fmt.Errorf("[Configuration] %+v", yamlErr)
		}
	}

	return nil
}

func InitRoutingConfigFactory(f string) error {
	if content, err := ioutil.ReadFile(f); err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	} else {
		UERoutingConfig = RoutingConfig{}

		if yamlErr := yaml.Unmarshal([]byte(content), &UERoutingConfig); yamlErr != nil {
			return fmt.Errorf("[Configuration] %+v", yamlErr)
		}
	}

	return nil

}

func GetSmfVersionInfo() string {
	return fmt.Sprintf("version is [%s], and expected is [%s].", SmfConfig.Info.Version, SMF_DEFAULT_VERSION)
}

func GetUeRoutingVersionInfo() string {
	return fmt.Sprintf("version is [%s], and expected is [%s].", UERoutingConfig.Info.Version, UE_ROUTING_DEFAULT_VERSION)
}

func CheckConfigVersion() error {

	if SmfConfig.Info.Version != "" {

		if err := checkVersionOverRequired(SmfConfig.Info.Version, SMF_DEFAULT_VERSION); err != nil {
			return fmt.Errorf("[Configuration] Current config version is [%s], but %+v", SmfConfig.Info.Version, err)
		}

		if err := checkVersionOverRequired(UERoutingConfig.Info.Version, UE_ROUTING_DEFAULT_VERSION); err != nil {
			return fmt.Errorf("[Configuration] Current config version is [%s], but %+v", UERoutingConfig.Info.Version, err)
		}

		return nil

	} else {
		return fmt.Errorf("[Configuration] Config without version field")
	}
}

// if versions are "1.0.1" and "1.0", the shorter will be widened to same length of the longer with zero

func checkVersionOverRequired(versionCkecked, versionRequired string) error {

	versionCkeckeds := strings.Split(versionCkecked, ".")
	versionRequireds := strings.Split(versionRequired, ".")

	maxLength := len(versionCkeckeds)

	if maxLength < len(versionRequired) {

		maxLength = len(versionRequired)
	}

	for i := 0; i < maxLength; i++ {

		var versionCkeckedNumber, versionRequiredNumber int
		var err error

		if len(versionCkeckeds) > i {

			if versionCkeckedNumber, err = strconv.Atoi(versionCkeckeds[i]); err != nil {
				return err
			}
		}

		if len(versionRequireds) > i {

			if versionRequiredNumber, err = strconv.Atoi(versionRequireds[i]); err != nil {
				return err
			}
		}

		if versionCkeckedNumber < versionRequiredNumber {
			return fmt.Errorf("expected config version is equal or over [%s]!!!", versionRequired)
		}

	}

	return nil
}
