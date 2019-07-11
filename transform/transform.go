package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

const (
	// MicrogatewayVersion is the version of the microgateway to use
	MicrogatewayVersion = "v0.0.0-20190708190753-c54f135979ec"
)

// Transform converts an asyn api to a new representation
func Transform(input, output, conversionType, role string) {
	switch role {
	case "server":
	case "client":
	default:
		panic("invalid role")
	}
	switch conversionType {
	case "flogoapiapp":
		ToAPI(input, output, role)
	case "flogodescriptor":
		ToJSON(input, output, role)
	default:
		panic("invalid type")
	}
}

type protocolConfig struct {
	name, secure                    string
	trigger, activity               string
	triggerImport, activityImport   string
	triggerVersion, activityVersion string
	port                            int
	contentPath                     string
	paramsPath                      string
	triggerSettings                 func(s settings) map[string]interface{}
	handlerSettings                 func(s settings) map[string]interface{}
	serviceSettings                 func(s settings) map[string]interface{}
}

var configs = [...]protocolConfig{
	protocolEFTL,
	protocolHTTP,
	protocolKafka,
	protocolMQTT,
	protocolWebsocket,
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
	parameters   []*models.Parameter
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

func (p protocolConfig) protocol(support *bytes.Buffer, model *models.AsyncapiDocument, schemes map[string]interface{}, flogo *app.Config, role string) {
	addImport := func(path, version string) {
		if version != "" {
			path = fmt.Sprintf(path, version)
		}
		for _, port := range flogo.Imports {
			if strings.Contains(port, path) {
				return
			}
		}
		flogo.Imports = append(flogo.Imports, path)
	}

	services, triggers := make([]*api.Service, 0, 8), make([]*trigger.Config, 0, 8)
	for i, server := range model.Servers {
		baseChannel := strings.TrimSuffix(server.BaseChannel, "/")
		if len(baseChannel) > 0 && !strings.HasPrefix(baseChannel, "/") {
			baseChannel = "/" + baseChannel
		}
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

			triggerVersion, activityVersion := s.triggerVersion, s.activityVersion
			if value := s.extensions["x-trigger-version"]; len(value) > 0 {
				var version string
				err := json.Unmarshal(value, &version)
				if err != nil {
					panic(err)
				}
				triggerVersion = version
			}
			if value := s.extensions["x-activity-version"]; len(value) > 0 {
				var version string
				err := json.Unmarshal(value, &version)
				if err != nil {
					panic(err)
				}
				activityVersion = version
			}
			addImport(p.triggerImport, triggerVersion)
			addImport(p.activityImport, activityVersion)

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
				s.parameters = channel.Parameters
				if strings.HasPrefix(name, "/") {
					s.topic = name
				} else {
					s.topic = baseChannel + "/" + name
				}
				subscribe, publish := channel.Subscribe, channel.Publish
				if role == "client" {
					subscribe, publish = publish, subscribe
				}
				if subscribe != nil {
					s.protocolInfo = nil
					if len(subscribe.ProtocolInfo) > 0 {
						err := json.Unmarshal(subscribe.ProtocolInfo, &s.protocolInfo)
						if err != nil {
							panic(err)
						}
					}
					handler := trigger.HandlerConfig{
						Settings: p.handlerSettings(s),
					}
					addImport("github.com/project-flogo/microgateway@%s", MicrogatewayVersion)
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
					if p.paramsPath != "" {
						actionConfig.Input["params"] = fmt.Sprintf("=$.%s", p.paramsPath)
					}
					handler.Actions = append(handler.Actions, &actionConfig)
					trig.Handlers = append(trig.Handlers, &handler)
				}
				if publish != nil && p.activity != "" {
					s.protocolInfo = nil
					if len(publish.ProtocolInfo) > 0 {
						err := json.Unmarshal(publish.ProtocolInfo, &s.protocolInfo)
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
			triggers = append(triggers, &trig)
		}
	}

	if len(triggers) > 0 {
		gateway := &api.Microgateway{
			Name: p.name,
		}
		addImport("github.com/project-flogo/contrib/activity/log", "")
		service := &api.Service{
			Name:        "log",
			Ref:         "github.com/project-flogo/contrib/activity/log",
			Description: "logging service",
		}
		gateway.Services = append(gateway.Services, service)
		addImport("github.com/nareshkumarthota/flogocomponents/activity/methodinvoker", "")
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
		flogo.Triggers = append(flogo.Triggers, triggers...)
	}

	if len(services) > 0 {
		addImport("github.com/project-flogo/contrib/trigger/rest", "")
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
		addImport("github.com/project-flogo/microgateway@%s", MicrogatewayVersion)
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
		addImport("github.com/project-flogo/contrib/activity/log", "")
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

func convert(input, role string) (*bytes.Buffer, *app.Config) {
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
		config.protocol(&support, &model, schemes, &flogo, role)
	}

	return &support, &flogo
}

// ToAPI converts an asyn api to a API flogo application
func ToAPI(input, output, role string) {
	support, flogo := convert(input, role)
	err := ioutil.WriteFile(output+"/support.go", support.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
	microgateway.Generate(flogo, output+"/app.go", output+"/go.mod")
}

// ToJSON converts an async api to a JSON flogo application
func ToJSON(input, output, role string) {
	support, flogo := convert(input, role)
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
