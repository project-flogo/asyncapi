package transform

import (
	"encoding/json"
	"fmt"

	"github.com/asyncapi/parser/pkg/models"
)

var protocolMQTT = protocolConfig{
	name:            "mqtt",
	secure:          "secure-mqtt",
	trigger:         "github.com/project-flogo/edge-contrib/trigger/mqtt",
	activity:        "github.com/project-flogo/edge-contrib/activity/mqtt",
	triggerImport:   "github.com/project-flogo/edge-contrib/trigger/mqtt@%s",
	activityImport:  "github.com/project-flogo/edge-contrib/activity/mqtt@%s",
	triggerVersion:  "v0.0.0-20190523234742-2d7b115b701a",
	activityVersion: "v0.0.0-20190523234742-2d7b115b701a",
	port:            9098,
	contentPath:     "message",
	paramsPath:      "topicParams",
	triggerSettings: func(s settings) map[string]interface{} {
		settings := map[string]interface{}{
			"id":     fmt.Sprintf("%s%d", s.name, s.serverIndex),
			"broker": s.url,
		}
		if s.userPassword {
			settings["username"] = s.user
			settings["password"] = s.password
		}
		if value := s.extensions["x-store"]; len(value) > 0 {
			var store string
			err := json.Unmarshal(value, &store)
			if err != nil {
				panic(err)
			}
			if store != "" {
				settings["store"] = store
			}
		}
		if value := s.extensions["x-clean-session"]; len(value) > 0 {
			var cleanSession bool
			err := json.Unmarshal(value, &cleanSession)
			if err != nil {
				panic(err)
			}
			settings["cleanSession"] = cleanSession
		}
		if value := s.extensions["x-keep-alive"]; len(value) > 0 {
			var keepAlive float64
			err := json.Unmarshal(value, &keepAlive)
			if err != nil {
				panic(err)
			}
			settings["keepAlive"] = keepAlive
		}
		if value := s.extensions["x-auto-reconnect"]; len(value) > 0 {
			var autoReconnect bool
			err := json.Unmarshal(value, &autoReconnect)
			if err != nil {
				panic(err)
			}
			settings["autoReconnect"] = autoReconnect
		}
		if s.secure {
			sslConfig := map[string]interface{}{
				"certFile": s.certFile,
				"keyFile":  s.keyFile,
			}
			skipVerify := true
			if value := s.extensions["x-skip-verify"]; len(value) > 0 {
				err := json.Unmarshal(value, &skipVerify)
				if err != nil {
					panic(err)
				}
				sslConfig["skipVerify"] = skipVerify
			}
			useSystemCert := true
			if value := s.extensions["x-use-systemcert"]; !skipVerify && len(value) > 0 {
				err := json.Unmarshal(value, &useSystemCert)
				if err != nil {
					panic(err)
				}
				sslConfig["useSystemCert"] = useSystemCert
			}
			if !useSystemCert {
				sslConfig["caFile"] = s.trustStore
			}
			settings["sslConfig"] = sslConfig
		}
		return settings
	},
	handlerSettings: func(s settings) map[string]interface{} {
		topic := s.topic[1:]
		settings := map[string]interface{}{
			"topic": topic,
		}
		chunks, hasVariables := parseURL(topic)
		if hasVariables {
			translated := ""
			for _, chunk := range chunks {
				if chunk.value != "" {
					translated += chunk.value
				} else {
					var parameter *models.Parameter
					for _, value := range s.parameters {
						if value.Name == chunk.name {
							parameter = value
							break
						}
					}
					if parameter != nil {
						if value := parameter.Extensions["x-multilevel"]; len(value) > 0 {
							var multilevel bool
							err := json.Unmarshal(value, &multilevel)
							if err != nil {
								panic(err)
							}
							if multilevel {
								translated += "#" + chunk.name
							} else {
								translated += "+" + chunk.name
							}
						}
					} else {
						translated += "+" + chunk.name
					}
				}
			}
			settings["topic"] = translated
		}
		if s.protocolInfo != nil {
			if value := s.protocolInfo["flogo-mqtt"]; value != nil {
				if mqtt, ok := value.(map[string]interface{}); ok {
					if value := mqtt["replyTopic"]; value != nil {
						if replyTopic, ok := value.(string); ok {
							settings["replyTopic"] = replyTopic
						}
					}
					if value := mqtt["qos"]; value != nil {
						if qos, ok := value.(float64); ok {
							settings["qos"] = int64(qos)
						}
					}
				}
			}
		}
		return settings
	},
	serviceSettings: func(s settings) map[string]interface{} {
		topic := s.topic[1:]
		settings := map[string]interface{}{
			"id":     fmt.Sprintf("%s%d_%s", s.name, s.serverIndex, s.topic),
			"broker": s.url,
			"topic":  topic,
		}
		chunks, hasVariables := parseURL(topic)
		if hasVariables {
			translated := ""
			for _, chunk := range chunks {
				if chunk.value != "" {
					translated += chunk.value
				} else {
					translated += ":" + chunk.name
				}
			}
			settings["topic"] = translated
		}
		if s.userPassword {
			settings["username"] = s.user
			settings["password"] = s.password
		}
		if s.protocolInfo != nil {
			if value := s.protocolInfo["flogo-mqtt"]; value != nil {
				if mqtt, ok := value.(map[string]interface{}); ok {
					if value := mqtt["store"]; value != nil {
						if store, ok := value.(string); ok {
							settings["store"] = store
						}
					}
					if value := mqtt["cleanSession"]; value != nil {
						if cleanSession, ok := value.(bool); ok {
							settings["cleanSession"] = cleanSession
						}
					}
					if value := mqtt["qos"]; value != nil {
						if qos, ok := value.(float64); ok {
							settings["qos"] = int64(qos)
						}
					}
					if s.secure {
						sslConfig := map[string]interface{}{
							"certFile": s.certFile,
							"keyFile":  s.keyFile,
						}
						skipVerify := true
						if value := mqtt["skipVerify"]; value != nil {
							if skipVerifyValue, ok := value.(bool); ok {
								settings["skipVerify"] = skipVerifyValue
								skipVerify = skipVerifyValue
							}
						}
						useSystemCert := true
						if value := mqtt["useSystemCert"]; !skipVerify && value != nil {
							if useSystemCertValue, ok := value.(bool); ok {
								settings["useSystemCert"] = useSystemCertValue
								useSystemCert = useSystemCertValue
							}
						}
						if !useSystemCert {
							sslConfig["caFile"] = s.trustStore
						}
						settings["sslConfig"] = sslConfig
					}
				}
			}
		}
		return settings
	},
}
