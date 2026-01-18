package postgres

type RestoreOptions struct {
	Host          string
	Port          int
	User          string
	Password      string
	Database      string
	Schema        string
	InputFile     string
	ParallelJobs  int
	StructureOnly bool
	DataOnly      bool
}

// RestoreStructure restores database structure
func RestoreStructure(opts RestoreOptions) error {
	// TODO: Implementation
	return nil
}

// RestoreData restores database data
func RestoreData(opts RestoreOptions) error {
	// TODO: Implementation
	return nil
}