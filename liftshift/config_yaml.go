package liftshift

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the structure of the YAML configuration file
type YAMLConfig struct {
	Source  SourceConfig  `yaml:"source"`
	Target  TargetConfig  `yaml:"target"`
	Options OptionsConfig `yaml:"options"`
}

type SourceConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type TargetConfig struct {
	Host          string   `yaml:"host"`
	Port          int      `yaml:"port"`
	Database      string   `yaml:"database"`
	AdminUser     string   `yaml:"admin_user"`
	AdminPassword string   `yaml:"admin_password"`
	AppUser       string   `yaml:"app_user"`
}

type OptionsConfig struct {
	ParallelJobs        int      `yaml:"parallel_jobs"`
	DataParallelJobs    int      `yaml:"data_parallel_jobs"`
	ExcludeTables       []string `yaml:"exclude_tables"`
	OutputDir           string   `yaml:"output_dir"`
	KeepDumps           bool     `yaml:"keep_dumps"`
	SkipBackup          bool     `yaml:"skip_backup"`
	TerminateConnections bool     `yaml:"terminate_connections"`
	Extensions          []string `yaml:"extensions"`
}

// LoadConfigFromYAML loads configuration from a YAML file
func LoadConfigFromYAML(filePath string) (*Config, error) {
	// Read the YAML file
	yamlData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Replace environment variables in the YAML content before unmarshaling
	processedYAML := substituteEnvVars(string(yamlData))

	// Parse the YAML into the YAMLConfig struct
	var yamlConfig YAMLConfig
	err = yaml.Unmarshal([]byte(processedYAML), &yamlConfig)
	if err != nil {
		return nil, err
	}

	// Convert YAMLConfig to Config
	config := &Config{
		SrcHost:          yamlConfig.Source.Host,
		SrcPort:          yamlConfig.Source.Port,
		SrcDB:            yamlConfig.Source.Database,
		SrcUser:          yamlConfig.Source.User,
		SrcPassword:      yamlConfig.Source.Password,
		DstHost:          yamlConfig.Target.Host,
		DstPort:          yamlConfig.Target.Port,
		DstDB:            yamlConfig.Target.Database,
		DstAdminUser:     yamlConfig.Target.AdminUser,
		DstAdminPassword: yamlConfig.Target.AdminPassword,
		AppUser:          yamlConfig.Target.AppUser,
		SkipBackup:       yamlConfig.Options.SkipBackup,
		ParallelJobs:     yamlConfig.Options.ParallelJobs,
		DataParallelJobs: yamlConfig.Options.DataParallelJobs,
		DryRun:           false, // Not specified in YAML, default to false
	}

	return config, nil
}

// substituteEnvVars replaces ${VAR_NAME} patterns with environment variable values
func substituteEnvVars(input string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract the variable name (without the ${} wrapper)
		varName := strings.TrimPrefix(match, "${")
		varName = strings.TrimSuffix(varName, "}")

		// Get the environment variable value
		value := os.Getenv(varName)

		// If the environment variable is not set, return the original match
		if value == "" && os.Getenv(varName) == "" {
			// Check if there's a default value specified like VAR_NAME:default_value
			parts := strings.SplitN(varName, ":", 2)
			if len(parts) == 2 {
				return parts[1] // Return the default value
			}
			return match // Return the original placeholder if no default
		}

		return value
	})
}

// LoadConfigWithDefaults loads configuration from YAML with reasonable defaults
func LoadConfigWithDefaults(filePath string) (*Config, error) {
	config, err := LoadConfigFromYAML(filePath)
	if err != nil {
		return nil, err
	}

	// Apply defaults for any zero values
	if config.ParallelJobs == 0 {
		config.ParallelJobs = 4
	}
	if config.DataParallelJobs == 0 {
		config.DataParallelJobs = 2
	}
	if config.SrcPort == 0 {
		config.SrcPort = 5432
	}
	if config.DstPort == 0 {
		config.DstPort = 5432
	}
	if config.SrcUser == "" {
		config.SrcUser = "postgres"
	}
	if config.DstAdminUser == "" {
		config.DstAdminUser = "postgres"
	}
	if config.AppUser == "" {
		config.AppUser = "app_user"
	}

	return config, nil
}