package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	parser "github.com/asyncapi/parser/pkg"
	_ "github.com/asyncapi/parser/pkg/errs"
	"github.com/asyncapi/parser/pkg/models"
	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/app"
	"github.com/project-flogo/core/app/resource"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/trigger"
	"github.com/project-flogo/microgateway"
	"github.com/project-flogo/microgateway/api"
)

// Transform converts an asyn api to a new representation
func Transform(input, output, conversionType string) {
	switch conversionType {
	case "flogoapiapp":
		ToAPI(input, output)
	case "flogodescriptor":
		ToJSON(input, output)
	default:
		panic("invalid type")
	}
}

type protocolConfig struct {
	name, secure      string
	trigger, activity string
	port              int
	contentPath       string
	triggerSettings   func(s settings) map[string]interface{}
	setTopic          func(s *settings, base, topic string)
	handlerSettings   func(s settings) map[string]interface{}
	serviceSettings   func(s settings) map[string]interface{}
}

type settings struct {
	protocolConfig
	secure       bool
	userPassword bool
	serverIndex  int
	url          string
	urlPort      string
	user         string
	password     string
	trustStore   string
	certFile     string
	keyFile      string
	extensions   map[string]json.RawMessage
	topic        string
	protocolInfo map[string]interface{}
}

func userPassword(server *models.Server, schemes map[string]interface{}) bool {
	for _, requirement := range server.Security {
		for scheme := range *requirement {
			if entry := schemes[scheme]; entry != nil {
				if definition, ok := entry.(map[string]interface{}); ok {
					if value := definition["type"]; value != nil {
						if typ, ok := value.(string); ok && typ == "userPassword" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

type chunk struct {
	name  string
	value string
}

func parseURL(url string) ([]chunk, bool) {
	var (
		chunks      []chunk
		parsed      []rune
		hasVariable bool
	)
	for _, s := range url {
		switch s {
		case '{':
			if len(parsed) > 0 {
				chunks, parsed = append(chunks, chunk{value: string(parsed)}), parsed[:0]
			}
		case '}':
			if len(parsed) > 0 {
				chunks, parsed = append(chunks, chunk{name: string(parsed)}), parsed[:0]
				hasVariable = true
			}
		default:
			parsed = append(parsed, s)
		}
	}
	if len(parsed) > 0 {
		chunks, parsed = append(chunks, chunk{value: string(parsed)}), parsed[:0]
	}
	return chunks, hasVariable
}

func getPort(url string) ([]chunk, bool) {
	var (
		chunks      []chunk
		parsed      []rune
		hasVariable bool
	)
	foundPort := false
	for _, s := range url {
		if foundPort {
			if s == '/' {
				foundPort = false
			} else {
				switch s {
				case '{':
					if len(parsed) > 0 {
						chunks, parsed = append(chunks, chunk{value: string(parsed)}), parsed[:0]
					}
				case '}':
					if len(parsed) > 0 {
						chunks, parsed = append(chunks, chunk{name: string(parsed)}), parsed[:0]
					}
					hasVariable = true
				default:
					parsed = append(parsed, s)
				}
			}
		} else if s == ':' {
			foundPort = true
		}
	}
	if len(parsed) > 0 {
		chunks, parsed = append(chunks, chunk{value: string(parsed)}), parsed[:0]
	}

	return chunks, hasVariable
}

func (p protocolConfig) protocol(support *bytes.Buffer, model *models.AsyncapiDocument, schemes map[string]interface{}, flogo *app.Config) {
	services := make([]*api.Service, 0, 8)
	for i, server := range model.Servers {
		if server.Protocol == p.name || server.Protocol == p.secure {
			if server.Variables != nil {
				for name, variable := range *server.Variables {
					defaultValue, foundDefault := variable.Default, false
					for j, value := range variable.Enum {
						if value == defaultValue {
							foundDefault = true
							attribute := data.NewAttribute(fmt.Sprintf("%s%d_%s", p.name, i, name), data.TypeString, value)
							flogo.Properties = append(flogo.Properties, attribute)
							continue
						}
						attribute := data.NewAttribute(fmt.Sprintf("%s%d_%s_%d", p.name, i, name, j), data.TypeString, value)
						flogo.Properties = append(flogo.Properties, attribute)
					}
					if !foundDefault {
						attribute := data.NewAttribute(fmt.Sprintf("%s%d_%s", p.name, i, name), data.TypeString, defaultValue)
						flogo.Properties = append(flogo.Properties, attribute)
					}
				}
			}

			brokerUrls := ""
			if chunks, hasVariable := parseURL(server.Url); hasVariable {
				if len(chunks) > 1 {
					comma := ""
					brokerUrls += "=string.concat("
					for _, chunk := range chunks {
						if chunk.name != "" {
							brokerUrls += fmt.Sprintf("%s$property[%s%d_%s]", comma, p.name, i, chunk.name)
							comma = ", "
							continue
						}
						brokerUrls += fmt.Sprintf("%s'%s'", comma, chunk.value)
						comma = ", "
					}
					brokerUrls += ")"
				} else {
					chunk := chunks[0]
					brokerUrls += "="
					brokerUrls += fmt.Sprintf("$property[%s%d_%s]", p.name, i, chunk.name)
				}
			} else {
				brokerUrls = fmt.Sprintf("%s%dURL", p.name, i)
				attribute := data.NewAttribute(brokerUrls, data.TypeString, server.Url)
				brokerUrls = fmt.Sprintf("=$property[%s]", brokerUrls)
				flogo.Properties = append(flogo.Properties, attribute)
			}

			s := settings{
				protocolConfig: p,
				secure:         server.Protocol == p.secure,
				userPassword:   userPassword(server, schemes),
				serverIndex:    i,
				url:            brokerUrls,
				user:           "=$env[USER]",
				password:       "=$env[PASSWORD]",
				trustStore:     "=$env[TRUST_STORE]",
				certFile:       "=$env[CERT_FILE]",
				keyFile:        "=$env[KEY_FILE]",
				extensions:     server.Extensions,
			}

			if chunks, hasVariable := getPort(server.Url); len(chunks) > 0 {
				if hasVariable {
					if len(chunks) > 1 {
						comma := ""
						s.urlPort += "=string.integer(string.concat("
						for _, chunk := range chunks {
							if chunk.name != "" {
								s.urlPort += fmt.Sprintf("%s$property[%s%d_%s]", comma, p.name, i, chunk.name)
								comma = ", "
								continue
							}
							s.urlPort += fmt.Sprintf("%s'%s'", comma, chunk.value)
							comma = ", "
						}
						s.urlPort += "))"
					} else {
						chunk := chunks[0]
						s.urlPort += "=string.integer("
						s.urlPort += fmt.Sprintf("$property[%s%d_%s]", p.name, i, chunk.name)
						s.urlPort += ")"
					}
				} else {
					port := ""
					for _, p := range chunks {
						port += p.value
					}
					value, err := strconv.Atoi(port)
					if err != nil {
						panic(err)
					}
					s.urlPort = fmt.Sprintf("%s%dPort", p.name, i)
					attribute := data.NewAttribute(s.urlPort, data.TypeInt, value)
					s.urlPort = fmt.Sprintf("=$property[%s]", s.urlPort)
					flogo.Properties = append(flogo.Properties, attribute)
				}
			}

			trig := trigger.Config{
				Id:       fmt.Sprintf("%s%d", p.name, i),
				Ref:      p.trigger,
				Settings: p.triggerSettings(s),
			}
			for name, channel := range model.Channels {
				if strings.HasPrefix(name, "/") {
					s.topic = name
				} else {
					p.setTopic(&s, server.BaseChannel, name)
				}
				if channel.Subscribe != nil {
					s.protocolInfo = nil
					if len(channel.Subscribe.ProtocolInfo) > 0 {
						err := json.Unmarshal(channel.Subscribe.ProtocolInfo, &s.protocolInfo)
						if err != nil {
							panic(err)
						}
					}
					handler := trigger.HandlerConfig{
						Settings: p.handlerSettings(s),
					}
					action := action.Config{
						Ref: "github.com/project-flogo/microgateway",
						Settings: map[string]interface{}{
							"uri":   fmt.Sprintf("microgateway:%s", p.name),
							"async": true,
						},
					}
					actionConfig := trigger.ActionConfig{
						Config: &action,
						Input: map[string]interface{}{
							"channel": fmt.Sprintf("='%s'", s.topic),
							"message": fmt.Sprintf("=$.%s", p.contentPath),
						},
					}
					handler.Actions = append(handler.Actions, &actionConfig)
					trig.Handlers = append(trig.Handlers, &handler)
				}
				if channel.Publish != nil && p.activity != "" {
					s.protocolInfo = nil
					if len(channel.Publish.ProtocolInfo) > 0 {
						err := json.Unmarshal(channel.Publish.ProtocolInfo, &s.protocolInfo)
						if err != nil {
							panic(err)
						}
					}
					service := &api.Service{
						Name:        fmt.Sprintf("%s-name-%s", p.name, name),
						Ref:         p.activity,
						Description: fmt.Sprintf("%s service", p.name),
						Settings:    p.serviceSettings(s),
					}
					services = append(services, service)
				}
			}
			flogo.Triggers = append(flogo.Triggers, &trig)
		}
	}

	if len(flogo.Triggers) > 0 {
		gateway := &api.Microgateway{
			Name: p.name,
		}
		service := &api.Service{
			Name:        "log",
			Ref:         "github.com/project-flogo/contrib/activity/log",
			Description: "logging service",
		}
		gateway.Services = append(gateway.Services, service)
		service = &api.Service{
			Name:        "methodinvoker",
			Ref:         "github.com/nareshkumarthota/flogocomponents/activity/methodinvoker",
			Description: "invoke a method",
		}
		gateway.Services = append(gateway.Services, service)
		step := &api.Step{
			Service: "log",
			Input: map[string]interface{}{
				"message": "=$.payload.message",
			},
		}
		gateway.Steps = append(gateway.Steps, step)
		step = &api.Step{
			Service: "methodinvoker",
			Input: map[string]interface{}{
				"methodName": fmt.Sprintf("%sMethod", p.name),
				"inputData":  "=$.payload",
			},
		}
		gateway.Steps = append(gateway.Steps, step)
		fmt.Fprintf(support, "func %sMethod(inputs interface{}) (map[string]interface{}, error) {\n", p.name)
		fmt.Fprintf(support, "\treturn nil, nil\n")
		fmt.Fprintf(support, "}\n")
		fmt.Fprintf(support, "func init() {\n")
		fmt.Fprintf(support, "\tmethodinvoker.RegisterMethods(\"%sMethod\", %sMethod)\n", p.name, p.name)
		fmt.Fprintf(support, "}\n")

		raw, err := json.Marshal(gateway)
		if err != nil {
			panic(err)
		}

		res := &resource.Config{
			ID:   fmt.Sprintf("microgateway:%s", p.name),
			Data: raw,
		}
		flogo.Resources = append(flogo.Resources, res)
	}

	if len(services) > 0 {
		trig := trigger.Config{
			Id:  fmt.Sprintf("%sPublish", p.name),
			Ref: "github.com/project-flogo/contrib/trigger/rest",
			Settings: map[string]interface{}{
				"port": p.port,
			},
		}
		handler := trigger.HandlerConfig{
			Settings: map[string]interface{}{
				"method": "POST",
				"path":   "/post",
			},
		}
		action := action.Config{
			Ref: "github.com/project-flogo/microgateway",
			Settings: map[string]interface{}{
				"uri":   fmt.Sprintf("microgateway:%sPublish", p.name),
				"async": true,
			},
		}
		actionConfig := trigger.ActionConfig{
			Config: &action,
		}
		handler.Actions = append(handler.Actions, &actionConfig)
		trig.Handlers = append(trig.Handlers, &handler)
		flogo.Triggers = append(flogo.Triggers, &trig)

		gateway := &api.Microgateway{
			Name: fmt.Sprintf("%sPublish", p.name),
		}
		service := &api.Service{
			Name:        "log",
			Ref:         "github.com/project-flogo/contrib/activity/log",
			Description: "logging service",
		}
		gateway.Services = append(services, service)
		step := &api.Step{
			Service: "log",
			Input: map[string]interface{}{
				"message": "=$.payload.content",
			},
		}
		gateway.Steps = append(gateway.Steps, step)

		raw, err := json.Marshal(gateway)
		if err != nil {
			panic(err)
		}

		res := &resource.Config{
			ID:   fmt.Sprintf("microgateway:%sPublish", p.name),
			Data: raw,
		}
		flogo.Resources = append(flogo.Resources, res)
	}
}

var configs = [...]protocolConfig{
	{
		name:        "kafka",
		secure:      "kafka-secure",
		trigger:     "github.com/project-flogo/contrib/trigger/kafka",
		activity:    "github.com/project-flogo/contrib/activity/kafka",
		port:        9096,
		contentPath: "message",
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
		setTopic: func(s *settings, base, topic string) {
			if base != "" {
				base = strings.TrimRight(strings.TrimLeft(base, "."), ".")
				topic = strings.TrimRight(strings.TrimLeft(topic, "."), ".")
				s.topic = fmt.Sprintf("%s.%s", base, topic)
				return
			}
			s.topic = topic
		},
		handlerSettings: func(s settings) map[string]interface{} {
			settings := map[string]interface{}{
				"topic": s.topic,
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
			settings := map[string]interface{}{
				"brokerUrls": s.url,
				"topic":      s.topic,
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
	},
	{
		name:        "eftl",
		secure:      "eftl-secure",
		trigger:     "github.com/project-flogo/eftl/trigger",
		activity:    "github.com/project-flogo/eftl/activity",
		port:        9097,
		contentPath: "content",
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
		setTopic: func(s *settings, base, topic string) {
			if base != "" {
				s.topic = fmt.Sprintf("%s_%s", base, topic)
				return
			}
			s.topic = topic
		},
		handlerSettings: func(s settings) map[string]interface{} {
			settings := map[string]interface{}{
				"dest": s.topic,
			}
			return settings
		},
		serviceSettings: func(s settings) map[string]interface{} {
			settings := map[string]interface{}{
				"id":   fmt.Sprintf("%s%s", s.name, s.topic),
				"url":  s.url,
				"dest": s.topic,
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
	},
	{
		name:        "mqtt",
		secure:      "secure-mqtt",
		trigger:     "github.com/project-flogo/edge-contrib/trigger/mqtt",
		activity:    "github.com/project-flogo/edge-contrib/activity/mqtt",
		port:        9098,
		contentPath: "message",
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
		setTopic: func(s *settings, base, topic string) {
			s.topic = path.Join(base, topic)
		},
		handlerSettings: func(s settings) map[string]interface{} {
			settings := map[string]interface{}{
				"topic": s.topic,
			}
			chunks, hasVariables := parseURL(s.topic)
			if hasVariables {
				translated := ""
				for _, chunk := range chunks {
					if chunk.value != "" {
						translated += chunk.value
					} else {
						translated += "+"
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
			settings := map[string]interface{}{
				"id":     fmt.Sprintf("%s%d_%s", s.name, s.serverIndex, s.topic),
				"broker": s.url,
				"topic":  s.topic,
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
	},
	{
		name:        "ws",
		secure:      "wss",
		trigger:     "github.com/project-flogo/websocket/trigger/wsclient",
		port:        9099,
		contentPath: "content",
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
		setTopic: func(s *settings, base, topic string) {
			if base != "" {
				s.topic = fmt.Sprintf("%s_%s", base, topic)
				return
			}
			s.topic = topic
		},
		handlerSettings: func(s settings) map[string]interface{} {
			settings := map[string]interface{}{}
			return settings
		},
	},
	{
		name:        "http",
		secure:      "https",
		trigger:     "github.com/project-flogo/contrib/trigger/rest",
		activity:    "github.com/project-flogo/contrib/activity/rest",
		port:        9100,
		contentPath: "content",
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
		setTopic: func(s *settings, base, topic string) {
			topic = path.Join(base, topic)
			if !strings.HasPrefix(topic, "/") {
				topic = "/" + topic
			}
			s.topic = topic
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
	},
}

func convert(input string) (*bytes.Buffer, *app.Config) {
	document, err := ioutil.ReadFile(input)
	if err != nil {
		panic(err)
	}

	parsed, perr := parser.Parse(document, true)
	if perr != nil {
		panic(err)
	}

	model := models.AsyncapiDocument{}
	err = json.Unmarshal(parsed, &model)
	if err != nil {
		panic(err)
	}

	flogo := app.Config{}
	flogo.Name = model.Id
	flogo.Type = "flogo:app"
	flogo.Version = "1.0.0"
	flogo.Description = model.Info.Description
	flogo.AppModel = "1.1.0"

	var schemes map[string]interface{}
	if len(model.Components.SecuritySchemes) > 0 {
		err = json.Unmarshal(model.Components.SecuritySchemes, &schemes)
		if err != nil {
			panic(err)
		}
	}

	support := bytes.Buffer{}
	fmt.Fprintf(&support, "package main\n")
	fmt.Fprintf(&support, "import \"github.com/nareshkumarthota/flogocomponents/activity/methodinvoker\"\n")
	for _, config := range configs {
		config.protocol(&support, &model, schemes, &flogo)
	}

	return &support, &flogo
}

// ToAPI converts an asyn api to a API flogo application
func ToAPI(input, output string) {
	support, flogo := convert(input)
	err := ioutil.WriteFile(output+"/support.go", support.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
	microgateway.Generate(flogo, output+"/app.go")
}

// ToJSON converts an async api to a JSON flogo application
func ToJSON(input, output string) {
	support, flogo := convert(input)
	err := ioutil.WriteFile(output+"/support.go", support.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
	data, err := json.MarshalIndent(flogo, "", "  ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(output+"/flogo.json", data, 0644)
	if err != nil {
		panic(err)
	}
}
