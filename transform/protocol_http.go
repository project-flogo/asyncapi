package transform

import (
	"fmt"
)

var protocolHTTP = protocolConfig{
	name:            "http",
	secure:          "https",
	trigger:         "github.com/project-flogo/contrib/trigger/rest",
	activity:        "github.com/project-flogo/contrib/activity/rest",
	triggerImport:   "github.com/project-flogo/contrib/trigger/rest@%s",
	activityImport:  "github.com/project-flogo/contrib/activity/rest@%s",
	triggerVersion:  "v0.9.0-rc.1.0.20190509204259-4246269fb68e",
	activityVersion: "v0.9.0-rc.1.0.20190509204259-4246269fb68e",
	port:            9100,
	contentPath:     "content",
	triggerSettings: func(s settings) map[string]interface{} {
		port := "80"
		if s.secure {
			port = "443"
		}
		if s.urlPort != "" {
			port = s.urlPort
		}
		settings := map[string]interface{}{
			"port": port,
		}
		if s.userPassword {
			// not supported
		}
		if s.secure {
			settings["enableTLS"] = true
			settings["certFile"] = s.certFile
			settings["keyFile"] = s.keyFile
		}
		return settings
	},
	handlerSettings: func(s settings) map[string]interface{} {
		settings := map[string]interface{}{
			"path": s.topic,
		}
		chunks, hasVariables := parseURL(s.topic)
		if hasVariables {
			translated := ""
			for _, chunk := range chunks {
				if chunk.value != "" {
					translated += chunk.value
				} else {
					translated += ":" + chunk.name
				}
			}
			settings["path"] = translated
		}

		if s.protocolInfo != nil {
			if value := s.protocolInfo["flogo-http"]; value != nil {
				if http, ok := value.(map[string]interface{}); ok {
					if value := http["method"]; value != nil {
						if method, ok := value.(string); ok {
							settings["method"] = method
						}
					}
				}
			}
		}
		return settings
	},
	serviceSettings: func(s settings) map[string]interface{} {
		path := s.topic
		chunks, hasVariables := parseURL(s.topic)
		if hasVariables {
			translated := ""
			for _, chunk := range chunks {
				if chunk.value != "" {
					translated += chunk.value
				} else {
					translated += ":" + chunk.name
				}
			}
			path = translated
		}
		settings := map[string]interface{}{
			"uri": fmt.Sprintf("=string.concat(%s, '%s')", s.url[1:], path),
		}

		if s.userPassword {
			// not supported
		}
		if s.protocolInfo != nil {
			if value := s.protocolInfo["flogo-http"]; value != nil {
				if http, ok := value.(map[string]interface{}); ok {
					if value := http["method"]; value != nil {
						if method, ok := value.(string); ok {
							settings["method"] = method
						}
					}
					if value := http["headers"]; value != nil {
						if headers, ok := value.(map[string]string); ok {
							settings["headers"] = headers
						}
					}
					if value := http["proxy"]; value != nil {
						if proxy, ok := value.(string); ok {
							settings["proxy"] = proxy
						}
					}
					if value := http["timeout"]; value != nil {
						if timeout, ok := value.(float64); ok {
							settings["timeout"] = int64(timeout)
						}
					}
					if s.secure {
						sslConfig := map[string]interface{}{
							"certFile": s.certFile,
							"keyFile":  s.keyFile,
						}
						skipVerify := true
						if value := http["skipVerify"]; value != nil {
							if skipVerifyValue, ok := value.(bool); ok {
								settings["skipVerify"] = skipVerifyValue
								skipVerify = skipVerifyValue
							}
						}
						useSystemCert := true
						if value := http["useSystemCert"]; !skipVerify && value != nil {
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
