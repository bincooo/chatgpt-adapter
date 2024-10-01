package vars

import (
	"chatgpt-adapter/logger"
	_ "embed"
	"encoding/json"
)

var (
	//go:embed js/dist/index.js
	Script string
)

type Config struct {
	PromptExperimentFirst string          `json:"PromptExperimentFirst"`
	PromptExperimentNext  string          `json:"PromptExperimentNext"`
	PersonalityFormat     string          `json:"PersonalityFormat"`
	ScenarioFormat        string          `json:"ScenarioFormat"`
	Settings              *ConfigSettings `json:"Settings,omitempty"`
}

type ConfigSettings struct {
	PromptExperiments bool `json:"PromptExperiments"`
	AllSamples        bool `json:"AllSamples"`
	NoSamples         bool `json:"NoSamples"`
	StripAssistant    bool `json:"StripAssistant"`
	StripHuman        bool `json:"StripHuman"`
	PassParams        bool `json:"PassParams"`
	ClearFlags        bool `json:"ClearFlags"`
	PreserveChats     bool `json:"PreserveChats"`
	FullColon         bool `json:"FullColon"`
	XmlPlot           bool `json:"xmlPlot"`
	SkipRestricted    bool `json:"SkipRestricted"`
}

type Replacements struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
	System    string `json:"system"`

	ExampleUser      string `json:"example_user"`
	ExampleAssistant string `json:"example_assistant"`
}

func ConvertConfig(config Config) (dict map[string]interface{}) {
	configBytes, err := json.Marshal(config)
	if err != nil {
		logger.Error(err)
		return
	}
	err = json.Unmarshal(configBytes, &dict)
	if err != nil {
		logger.Error(err)
		return
	}

	for _, key := range []string{
		"PromptExperimentFirst",
		"PromptExperimentNext",
		"PersonalityFormat",
		"ScenarioFormat",
	} {
		if dict[key] == "" {
			delete(dict, key)
		}
	}

	return
}

func ConvertReplacements(replacements Replacements) (dict map[string]interface{}) {
	replacementsBytes, err := json.Marshal(replacements)
	if err != nil {
		logger.Error(err)
		return
	}
	err = json.Unmarshal(replacementsBytes, &dict)
	if err != nil {
		logger.Error(err)
		return
	}

	for key, value := range map[string]string{
		"user":              "Human",
		"assistant":         "Assistant",
		"example_user":      "H",
		"example_assistant": "A",
		"system":            "",
	} {
		if dict[key] == "" {
			dict[key] = value
		}
	}

	return
}
