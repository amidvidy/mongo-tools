package mongorestore

import (
	"fmt"
	"github.com/mongodb/mongo-tools/common/db"
	"github.com/mongodb/mongo-tools/common/log"
	commonopts "github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/mongorestore/options"
	"gopkg.in/mgo.v2"
)

type MongoRestore struct {
	ToolOptions   *commonopts.ToolOptions
	InputOptions  *options.InputOptions
	OutputOptions *options.OutputOptions

	cmdRunner db.CommandRunner

	TargetDirectory string

	// other internal state
	manager  *IntentManager
	safety   *mgo.Safe
	objCheck bool
}

func (restore *MongoRestore) Init() error {
	if restore.ToolOptions.Namespace.DBPath != "" {
		shim, err := db.NewShim(restore.ToolOptions.Namespace.DBPath, restore.ToolOptions.DirectoryPerDB, restore.ToolOptions.Journal)
		if err != nil {
			return err
		}
		fmt.Printf("%#v", shim)
		restore.cmdRunner = shim
		return nil
	}
	restore.cmdRunner = db.NewSessionProvider(*restore.ToolOptions)
	return nil
}

func (restore *MongoRestore) ParseAndValidateOptions() error {
	// Can't use option pkg defaults for --objcheck because it's two seperate flags,
	// and we need to be able to see if they're both being used. We default to
	// true here and then see if noobjcheck is enable.
	log.Log(3, "checking options")
	restore.objCheck = true
	if restore.InputOptions.NoObjcheck {
		restore.objCheck = false
		log.Log(3, "\tdumping with object check disabled")
		if restore.InputOptions.Objcheck {
			return fmt.Errorf("cannot use both the --objcheck and --noobjcheck flags")
		}
	} else {
		log.Log(3, "\tdumping with object check enabled")
	}

	if restore.OutputOptions.WriteConcern > 0 {
		restore.safety = &mgo.Safe{W: restore.OutputOptions.WriteConcern} //TODO, audit extra steps
		log.Logf(3, "\tdumping with w=%v", restore.safety.W)
	}

	//TODO check oplog is okay

	return nil
}

func (restore *MongoRestore) Restore() error {
	// TODO validate options
	err := restore.ParseAndValidateOptions()
	if err != nil {
		return fmt.Errorf("options error: %v", err)
	}

	// 1. Build up all intents to be restored
	restore.manager = NewIntentManager()

	switch {
	case restore.ToolOptions.DB == "" && restore.ToolOptions.Collection == "":
		log.Logf(0,
			"building a list of dbs and collections to restore from %v dir",
			restore.TargetDirectory)
		err = restore.CreateAllIntents(restore.TargetDirectory)
	case restore.ToolOptions.DB != "" && restore.ToolOptions.Collection == "":
		log.Logf(0,
			"building a list of collections to restore from %v dir",
			restore.TargetDirectory)
		err = restore.CreateIntentsForDB(
			restore.ToolOptions.DB,
			restore.TargetDirectory)
	case restore.ToolOptions.DB != "" && restore.ToolOptions.Collection != "":
		log.Logf(0, "checking for collection data in %v", restore.TargetDirectory)
		err = restore.CreateIntentForCollection(
			restore.ToolOptions.DB,
			restore.ToolOptions.Collection,
			restore.TargetDirectory)
	}
	if err != nil {
		return fmt.Errorf("error scanning filesystem: %v", err)
	}

	// 2. Restore them...
	err = restore.RestoreIntents()
	if err != nil {
		return fmt.Errorf("restore error: %v", err)
	}

	// 3. Restore oplog
	if restore.InputOptions.OplogReplay {
		err = restore.RestoreOplog()
		if err != nil {
			return fmt.Errorf("restore error: %v", err)
		}
	}
	log.Log(0, "done")

	return nil
}
