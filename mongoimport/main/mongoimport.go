package main

import (
	"fmt"
	"github.com/mongodb/mongo-tools/common/db"
	"github.com/mongodb/mongo-tools/common/log"
	commonopts "github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/common/util"
	"github.com/mongodb/mongo-tools/mongoimport"
	"github.com/mongodb/mongo-tools/mongoimport/options"
)

func main() {
	// initialize command-line opts
	usageStr := " --host myhost --db my_cms --collection docs < mydocfile." +
		"json \n\nImport CSV, TSV or JSON data into MongoDB.\n\nWhen importing " +
		"JSON documents, each document must be a separate line of the input file."
	opts := commonopts.New("mongoimport", usageStr)

	inputOpts := &options.InputOptions{}
	opts.AddOptions(inputOpts)
	ingestOpts := &options.IngestOptions{}
	opts.AddOptions(ingestOpts)

	args, err := opts.Parse()
	if err != nil {
		log.Logf(log.Always, "error parsing command line options: %v", err)
		util.ExitFail()
	}

	log.SetVerbosity(opts.Verbosity)

	// print help, if specified
	if opts.PrintHelp(false) {
		return
	}

	// print version, if specified
	if opts.PrintVersion() {
		return
	}

	// don't attempt to discover other members of a replica set
	opts.Direct = true

	// create a session provider to connect to the db
	sessionProvider := db.NewSessionProvider(*opts)

	mongoImport := mongoimport.MongoImport{
		ToolOptions:     opts,
		InputOptions:    inputOpts,
		IngestOptions:   ingestOpts,
		SessionProvider: sessionProvider,
	}

	if err = mongoImport.ValidateSettings(args); err != nil {
		log.Logf(log.Always, "error validating settings: %v", err)
		util.ExitFail()
	}

	numDocs, err := mongoImport.ImportDocuments()
	if !opts.Quiet {
		if err != nil {
			log.Logf(log.Always, "error(s) encountered while importing documents: %v", err)
		}
		message := fmt.Sprintf("imported 1 document")
		if numDocs != 1 {
			message = fmt.Sprintf("imported %v documents", numDocs)
		}
		log.Logf(log.Always, message)
	}
	if err != nil {
		util.ExitFail()
	}
}
