package transform

import (
	"strings"
)

var protocolWebsocket = protocolConfig{
	name:           "ws",
	secure:         "wss",
	trigger:        "github.com/project-flogo/websocket/trigger/wsclient",
	triggerImport:  "github.com/project-flogo/websocket@%s:/trigger/wsclient",
	triggerVersion: "v0.0.0-20190708195807-1d89e706e274",
	port:           9099,
	contentPath:    "content",
	triggerSettings: func(s settings) map[string]interface{} {
		settings := map[string]interface{}{
			"url": s.url,
		}
		if s.userPassword {
			// not supported
		}
		if s.secure {
			// supproted
		}
		return settings
	},
	handlerSettings: func(s settings) map[string]interface{} {
		parts := strings.Split(s.topic[1:], "/")
		topic := strings.Join(parts, "_")
		_ = topic
		settings := map[string]interface{}{}
		return settings
	},
}
