package transform

import (
	"fmt"
	"strings"
)

var protocolEFTL = protocolConfig{
	name:            "eftl",
	secure:          "eftl-secure",
	trigger:         "github.com/project-flogo/eftl/trigger",
	activity:        "github.com/project-flogo/eftl/activity",
	triggerImport:   "github.com/project-flogo/eftl@%s:/trigger",
	activityImport:  "github.com/project-flogo/eftl@%s:/activity",
	triggerVersion:  "v0.0.0-20190709194620-9c397d37ddf5",
	activityVersion: "v0.0.0-20190709194620-9c397d37ddf5",
	port:            9097,
	contentPath:     "content",
	triggerSettings: func(s settings) map[string]interface{} {
		settings := map[string]interface{}{
			"id":  fmt.Sprintf("%s%d", s.name, s.serverIndex),
			"url": s.url,
		}
		if s.userPassword {
			settings["user"] = s.user
			settings["password"] = s.password
		}
		if s.secure {
			settings["ca"] = s.trustStore
		}
		return settings
	},
	handlerSettings: func(s settings) map[string]interface{} {
		parts := strings.Split(s.topic[1:], "/")
		topic := strings.Join(parts, "_")
		settings := map[string]interface{}{
			"dest": topic,
		}
		return settings
	},
	serviceSettings: func(s settings) map[string]interface{} {
		parts := strings.Split(s.topic[1:], "/")
		topic := strings.Join(parts, "_")
		settings := map[string]interface{}{
			"id":   fmt.Sprintf("%s%s", s.name, s.topic),
			"url":  s.url,
			"dest": topic,
		}
		if s.userPassword {
			settings["user"] = s.user
			settings["password"] = s.password
		}
		if s.secure {
			settings["ca"] = s.trustStore
		}
		return settings
	},
}
