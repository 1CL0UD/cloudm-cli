package postgres

type DumpOptions struct {
	Host          string
	Port          int
	User          string
	Password      string
	Database      string
	Schema        string
	OutputFile    string
	StructureOnly bool
	DataOnly      bool
	ExcludeTables []string
}

// DumpStructure dumps database structure
func DumpStructure(opts DumpOptions) error {
	// TODO: Implementation
	return nil
}

// DumpData dumps database data
func DumpData(opts DumpOptions) error {
	// TODO: Implementation
	return nil
}