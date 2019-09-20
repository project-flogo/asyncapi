package transform

import (
	"fmt"

	"github.com/project-flogo/asyncapi/transform/models"
)

var protocolMQTT = protocolConfig{
	name:            "mqtt",
	secure:          "secure-mqtt",
	trigger:         "github.com/project-flogo/edge-contrib/trigger/mqtt",
	activity:        "github.com/project-flogo/edge-contrib/activity/mqtt",
	triggerImport:   "github.com/project-flogo/edge-contrib/trigger/mqtt@%s",
	activityImport:  "github.com/project-flogo/edge-contrib/activity/mqtt@%s",
	triggerVersion:  "v0.0.0-20190711193600-08aa43fa8ef4",
	activityVersion: "v0.0.0-20190711193600-08aa43fa8ef4",
	port:            9098,
	contentPath:     "message",
	paramsPath:      "topicParams",
	triggerSettings: func(s settings) map[string]interface{} {
		settings := map[string]interface{}{
			"id":     fmt.Sprintf("%s%s", s.name, s.serverName),
			"broker": s.url,
		}
		if s.userPassword {
			settings["username"] = s.user
			settings["password"] = s.password
		}
		if value, ok := s.extensions["x-store"]; ok {
			if store, ok := value.(string); ok {
				if store != "" {
					settings["store"] = store
				}
			}
		}
		if value, ok := s.extensions["x-clean-session"]; ok {
			if cleanSession, ok := value.(bool); ok {
				settings["cleanSession"] = cleanSession
			}
		}
		if value, ok := s.extensions["x-keep-alive"]; ok {
			if keepAlive, ok := value.(float64); ok {
				settings["keepAlive"] = keepAlive
			}
		}
		if value, ok := s.extensions["x-auto-reconnect"]; ok {
			if autoReconnect, ok := value.(bool); ok {
				settings["autoReconnect"] = autoReconnect
			}
		}
		if s.secure {
			sslConfig := map[string]interface{}{
				"certFile": s.certFile,
				"keyFile":  s.keyFile,
			}
			skipVerify := true
			if value, ok := s.extensions["x-skip-verify"]; ok {
				if value, ok := value.(bool); ok {
					skipVerify = value
					sslConfig["skipVerify"] = skipVerify
				}
			}
			useSystemCert := true
			if value, ok := s.extensions["x-use-systemcert"]; ok {
				if value, ok := value.(bool); !skipVerify && ok {
					useSystemCert = value
					sslConfig["useSystemCert"] = useSystemCert
				}
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
					for name, value := range s.parameters {
						if name == chunk.name {
							parameter = value
							break
						}
					}
					if parameter != nil {
						if value, ok := parameter.AdditionalProperties["x-multilevel"]; ok {
							if multilevel, ok := value.(bool); ok {
								if multilevel {
									translated += "#" + chunk.name
								} else {
									translated += "+" + chunk.name
								}
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
			"id":     fmt.Sprintf("%s%s_%s", s.name, s.serverName, s.topic),
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
