package transform

import (
	"strings"
)

var protocolKafka = protocolConfig{
	name:            "kafka",
	secure:          "kafka-secure",
	trigger:         "github.com/project-flogo/contrib/trigger/kafka",
	activity:        "github.com/project-flogo/contrib/activity/kafka",
	triggerImport:   "github.com/project-flogo/contrib/trigger/kafka@%s",
	activityImport:  "github.com/project-flogo/contrib/activity/kafka@%s",
	triggerVersion:  "v0.9.1-0.20190603184501-d845e1d612f8",
	activityVersion: "v0.9.1-0.20190516180541-534215f1b7ac",
	port:            9096,
	contentPath:     "message",
	triggerSettings: func(s settings) map[string]interface{} {
		settings := map[string]interface{}{
			"brokerUrls": s.url,
		}
		if s.userPassword {
			settings["user"] = s.user
			settings["password"] = s.password
		}
		if s.secure {
			settings["trustStore"] = s.trustStore
		}
		return settings
	},
	handlerSettings: func(s settings) map[string]interface{} {
		parts := strings.Split(s.topic[1:], "/")
		topic := strings.Join(parts, ".")
		settings := map[string]interface{}{
			"topic": topic,
		}
		if s.protocolInfo != nil {
			if value := s.protocolInfo["flogo-kafka"]; value != nil {
				if flogo, ok := value.(map[string]interface{}); ok {
					if value := flogo["partitions"]; value != nil {
						if partitions, ok := value.(string); ok {
							settings["partitions"] = partitions
						}
					}
					if value := flogo["offset"]; value != nil {
						if offset, ok := value.(float64); ok {
							settings["offset"] = int64(offset)
						}
					}
				}
			}
		}
		return settings
	},
	serviceSettings: func(s settings) map[string]interface{} {
		parts := strings.Split(s.topic[1:], "/")
		topic := strings.Join(parts, ".")
		settings := map[string]interface{}{
			"brokerUrls": s.url,
			"topic":      topic,
		}
		if s.userPassword {
			settings["user"] = s.user
			settings["password"] = s.password
		}
		if s.secure {
			settings["trustStore"] = s.trustStore
		}
		return settings
	},
}
