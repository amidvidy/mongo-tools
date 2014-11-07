package mongorestore

import (
	"fmt"
	"github.com/mongodb/mongo-tools/common/db"
	"github.com/mongodb/mongo-tools/common/intents"
	"github.com/mongodb/mongo-tools/common/log"
	commonopts "github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/common/progress"
	"github.com/mongodb/mongo-tools/mongorestore/options"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strconv"
)

type MongoRestore struct {
	ToolOptions   *commonopts.ToolOptions
	InputOptions  *options.InputOptions
	OutputOptions *options.OutputOptions

	SessionProvider *db.SessionProvider

	TargetDirectory string

	tempUsersCol string
	tempRolesCol string

	// other internal state
	manager         *intents.Manager
	safety          *mgo.Safe
	progressManager *progress.Manager
	objCheck        bool
	oplogLimit      bson.MongoTimestamp
}

func (restore *MongoRestore) ParseAndValidateOptions() error {
	// Can't use option pkg defaults for --objcheck because it's two separate flags,
	// and we need to be able to see if they're both being used. We default to
	// true here and then see if noobjcheck is enable.
	log.Log(log.DebugHigh, "checking options")
	if err := restore.ToolOptions.Validate(); err != nil {
		return err
	}
	restore.objCheck = true
	if restore.InputOptions.NoObjcheck {
		restore.objCheck = false
		log.Log(log.DebugHigh, "\tdumping with object check disabled")
		if restore.InputOptions.Objcheck {
			return fmt.Errorf("cannot use both the --objcheck and --noobjcheck flags")
		}
	} else {
		log.Log(log.DebugHigh, "\tdumping with object check enabled")
	}

	if restore.ToolOptions.DB == "" && restore.ToolOptions.Collection != "" {
		return fmt.Errorf("cannot dump a collection without a specified database")
	}

	if restore.InputOptions.OplogLimit != "" {
		if !restore.InputOptions.OplogReplay {
			return fmt.Errorf("cannot use --oplogLimit without --oplogReplay enabled")
		}
		var err error
		restore.oplogLimit, err = ParseTimestampFlag(restore.InputOptions.OplogLimit)
		if err != nil {
			return fmt.Errorf("error parsing timestamp argument to --oplogLimit: %v", err)
		}
	}

	if restore.OutputOptions.WriteConcern == "" || restore.OutputOptions.WriteConcern == "majority" {
		log.Logf(log.DebugLow, "\tdumping with w=majority")

		// check if we are using a replica set and fall back to w=1 if we aren't (for <= 2.4)
		isRepl, err := restore.SessionProvider.IsReplicaSet()
		if err != nil {
			return fmt.Errorf("error determining if connected to replica set: %v", err)
		}

		if isRepl {
			restore.safety = &mgo.Safe{WMode: "majority"}
		} else {
			log.Logf(log.DebugHigh, "\t\tnot connected to a replset, using equivalent w=1 for backward compatibility")
			restore.safety = &mgo.Safe{W: 1}
		}
	} else {
		intWriteConcern, err := strconv.Atoi(restore.OutputOptions.WriteConcern)
		if err != nil {
			return fmt.Errorf("error parsing --w value: %v", err)
		}
		if intWriteConcern < 0 {
			return fmt.Errorf("cannot use a negative write concern")
		}

		log.Logf(log.DebugLow, "\tdumping with w=%v", intWriteConcern)
		if intWriteConcern == 0 {
			// we must set safety to nil for true fire-and-forget behavior
			restore.safety = nil
		} else {
			restore.safety = &mgo.Safe{W: intWriteConcern}
		}
	}

	if restore.tempUsersCol == "" {
		restore.tempUsersCol = "tempusers"
	}
	if restore.tempRolesCol == "" {
		restore.tempRolesCol = "temproles"
	}

	if restore.OutputOptions.BulkWriters < 0 {
		return fmt.Errorf(
			"cannot specify a negative number of insertion workers per collection")
	}

	return nil
}

func (restore *MongoRestore) Restore() error {
	err := restore.ParseAndValidateOptions()
	if err != nil {
		return fmt.Errorf("options error: %v", err)
	}

	// 1. Build up all intents to be restored
	restore.manager = intents.NewCategorizingIntentManager()

	switch {
	case restore.ToolOptions.DB == "" && restore.ToolOptions.Collection == "":
		log.Logf(log.Always,
			"building a list of dbs and collections to restore from %v dir",
			restore.TargetDirectory)
		err = restore.CreateAllIntents(restore.TargetDirectory)
	case restore.ToolOptions.DB != "" && restore.ToolOptions.Collection == "":
		log.Logf(log.Always,
			"building a list of collections to restore from %v dir",
			restore.TargetDirectory)
		err = restore.CreateIntentsForDB(
			restore.ToolOptions.DB,
			restore.TargetDirectory)
	case restore.ToolOptions.DB != "" && restore.ToolOptions.Collection != "":
		log.Logf(log.Always, "checking for collection data in %v", restore.TargetDirectory)
		err = restore.CreateIntentForCollection(
			restore.ToolOptions.DB,
			restore.ToolOptions.Collection,
			restore.TargetDirectory)
	}
	if err != nil {
		return fmt.Errorf("error scanning filesystem: %v", err)
	}

	// 2. Restore them...
	if restore.OutputOptions.JobThreads > 0 {
		restore.manager.Finalize(intents.MultiDatabaseLTF)
	} else {
		// use legacy restoration order if we are single-threaded
		restore.manager.Finalize(intents.Legacy)
	}
	err = restore.RestoreIntents()
	if err != nil {
		return fmt.Errorf("restore error: %v", err)
	}

	// 3. Restore users/roles
	// TODO comment all cases
	if restore.InputOptions.RestoreDBUsersAndRoles || restore.ToolOptions.DB == "" || restore.ToolOptions.DB == "admin" {
		if restore.manager.Users() != nil {
			err = restore.RestoreUsersOrRoles(Users, restore.manager.Users())
			if err != nil {
				return fmt.Errorf("restore error: %v", err)
			}
		}
		if restore.manager.Roles() != nil {
			err = restore.RestoreUsersOrRoles(Roles, restore.manager.Roles())
			if err != nil {
				return fmt.Errorf("restore error: %v", err)
			}
		}
	}

	// 4. Restore oplog
	if restore.InputOptions.OplogReplay {
		err = restore.RestoreOplog()
		if err != nil {
			return fmt.Errorf("restore error: %v", err)
		}
	}

	log.Log(log.Always, "done")
	return nil
}
