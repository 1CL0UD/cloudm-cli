package config

type Config struct {
	Source  DatabaseConfig   `yaml:"source"`
	Target  TargetConfig     `yaml:"target"`
	Options MigrationOptions `yaml:"options"`
}

type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Database     string `yaml:"database"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	PasswordFile string `yaml:"password_file"`
}

type TargetConfig struct {
	DatabaseConfig  `yaml:",inline"`
	AdminUser       string `yaml:"admin_user"`
	AdminPassword   string `yaml:"admin_password"`
	AppUser         string `yaml:"app_user"`
	AppUserPassword string `yaml:"app_user_password"`
}

type MigrationOptions struct {
	ParallelJobs     int      `yaml:"parallel_jobs"`
	DataParallelJobs int      `yaml:"data_parallel_jobs"`
	ExcludeTables    []string `yaml:"exclude_tables"`
	OutputDir        string   `yaml:"output_dir"`
	KeepDumps        bool     `yaml:"keep_dumps"`
	SkipBackup       bool     `yaml:"skip_backup"`
	TerminateConns   bool     `yaml:"terminate_connections"`
	Extensions       []string `yaml:"extensions"`
}