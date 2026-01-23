package liftshift

// Config holds the configuration for the cloudm-cli application
type Config struct {
	SrcHost          string
	SrcPort          int
	SrcDB            string
	SrcUser          string
	SrcPassword      string
	DstHost          string
	DstPort          int
	DstDB            string
	DstAdminUser     string
	DstAdminPassword string
	AppUser          string
	SkipBackup       bool
	ParallelJobs     int
	DataParallelJobs int
	DryRun           bool
}
