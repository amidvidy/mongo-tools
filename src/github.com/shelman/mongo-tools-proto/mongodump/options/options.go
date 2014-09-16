package options

//TODO audit descriptions

type InputOptions struct {
	Query     string `long:"query" short:"q" description:"query filter, as a JSON string, e.g., '{x:{$gt:1}}'"`
	TableScan bool   `long:"forceTableScan" description:"force a table scan"`
	//SlaveOk
}

func (self *InputOptions) Name() string {
	return "query"
}

type OutputOptions struct {
	Out                 string `long:"out" short:"o" description:"output directory or - for stdout" default:"dump"`
	Oplog               bool   `long:"oplog" description:"Use oplog for point-in-time snapshotting"`
	DumpDBUsersAndRoles bool   `long:"dumpDbUsersAndRoles" description:"Dump user and role definitions for the given database"`
}

func (self *OutputOptions) Name() string {
	return "output"
}
