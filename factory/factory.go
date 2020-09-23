/*
 * AMF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"

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
