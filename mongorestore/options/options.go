package options

type InputOptions struct {
	Objcheck               bool   `long:"objcheck" description:"Validate object before inserting (default)"`
	NoObjcheck             bool   `long:"noobjcheck" description:"Don't validate object before inserting"`
	OplogReplay            bool   `long:"oplogReplay" description:"Replay oplog for point-in-time restore"`
	OplogLimit             string `long:"oplogLimit" description:"Include oplog entries before the provided Timestamp (seconds[:ordinal])"`
	RestoreDBUsersAndRoles bool   `long:"restoreDbUsersAndRoles" description:"Restore user and role definitions for the given database"`
	Directory              string `long:"dir" description:"alternative flag for entering the dump directory"`
}

func (self *InputOptions) Name() string {
	return "input"
}

type OutputOptions struct {
	Drop             bool   `long:"drop" description:"Drop each collection before import"`
	WriteConcern     string `long:"w" default:"majority" description:"Write concern options e.g. --w majority, --w '{w: 3, wtimeout: 500, fsync: true, j: true}'"`
	NoIndexRestore   bool   `long:"noIndexRestore" description:"Don't restore indexes"`
	NoOptionsRestore bool   `long:"noOptionsRestore" description:"Don't restore options"`
	KeepIndexVersion bool   `long:"keepIndexVersion" description:"Don't update index version"`

	JobThreads       int  `long:"numParallelCollections" short:"j" description:"Number of collections to restore in parallel" default:"4"`
	BulkWriters      int  `long:"numInsertionWorkersPerCollection" description:"Number of insert connections per collection" default:"1"`
	BulkBufferSize   int  `long:"batchSize" description:"Maximum number of documents to coalesce into a single bulk insertion" default:"10000"`
	PreserveDocOrder bool `long:"preserveOrder" description:"Preserve order of documents during restoration"`
	// TODO: add hidden option for NumOSThreads to set GOMAXPROCS on CLI
}

func (self *OutputOptions) Name() string {
	return "restore"
}
