package util

import (
	"fmt"
	"github.com/mongodb/mongo-tools/common/json"
	"github.com/mongodb/mongo-tools/common/log"
	"gopkg.in/mgo.v2"
	"strconv"
)

// write concern fields
const (
	J        = "j"
	W        = "w"
	FSYNC    = "fsync"
	WTIMEOUT = "wtimeout"
)

// getBoolArgument takes in an argument name an interface value and attempts to type
// assert or convert the value to an boolean. It returns an error if it fails to
// retrieve an boolean from the value
func getBoolArgument(name string, v interface{}) (bool, error) {
	switch value := v.(type) {
	case bool:
		return value, nil
	case string:
		return strconv.ParseBool(value)
	}
	return false, fmt.Errorf("invalid %v argument: %v", name, v)
}

// getStringArgument takes in an argument name an interface value and attempts to type
// assert or convert the value to an string. It returns an error if it fails to
// retrieve an string from the value
func getStringArgument(name string, v interface{}) (string, error) {
	if value, ok := v.(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("invalid %v argument: %v", name, v)
}

// getIntArgument takes in an argument name an interface value and attempts to type
// assert or convert the value to an int. It returns an error if it fails to
// retrieve an int from the value
func getIntArgument(name string, v interface{}) (int, error) {
	switch value := v.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		return int(value), nil
	case string:
		intValue, err := strconv.Atoi(value)
		if err == nil {
			return intValue, nil
		}
	}
	return 0, fmt.Errorf("invalid %v argument: %v", name, v)
}

// constructWCObject takes in a write concern and attempts to construct an
// mgo.Safe object from it. It returns an error if it is unable to parse the
// string or if a parsed write concern field value is invalid.
func constructWCObject(writeConcern string) (sessionSafety *mgo.Safe, err error) {
	sessionSafety = &mgo.Safe{}
	defer func() {
		// If the user passes a w value of 0, we set the session to use the
		// unacknowledged write concern but only if journal commit acknowledgment,
		// is not required. If commit acknowledgment is required, it prevails,
		// and the server will require that mongod acknowledge the write operation
		if sessionSafety.WMode == "" && sessionSafety.W == 0 && !sessionSafety.J {
			sessionSafety = nil
		}
	}()
	jsonWriteConcern := map[string]interface{}{}

	if err = json.Unmarshal([]byte(writeConcern), &jsonWriteConcern); err != nil {
		// if the writeConcern string can not be unmarshaled into JSON, this
		// allows a default to the old behavior wherein the entire argument
		// passed in is assigned to the 'w' field - thus allowing users pass
		// a write concern that looks like: "majority", 0, "4", etc.
		wValue, err := getIntArgument(W, writeConcern)
		if err != nil {
			// check if it's a string, if not, error out
			wStrVal, err := getStringArgument(W, writeConcern)
			if err != nil {
				return sessionSafety, err
			}
			sessionSafety.WMode = wStrVal
		} else {
			sessionSafety.W = wValue
		}
		return sessionSafety, nil
	}

	if j, ok := jsonWriteConcern[J]; ok {
		jValue, err := getBoolArgument(J, j)
		if err != nil {
			return sessionSafety, err
		}
		sessionSafety.J = jValue
	}

	if fsync, ok := jsonWriteConcern[FSYNC]; ok {
		fsyncValue, err := getBoolArgument(FSYNC, fsync)
		if err != nil {
			return sessionSafety, err
		}
		sessionSafety.FSync = fsyncValue
	}

	if wtimeout, ok := jsonWriteConcern[WTIMEOUT]; ok {
		wtimeoutValue, err := getIntArgument(WTIMEOUT, wtimeout)
		if err != nil {
			return sessionSafety, err
		}
		sessionSafety.WTimeout = wtimeoutValue
	}

	if w, ok := jsonWriteConcern[W]; ok {
		wValue, err := getIntArgument(W, w)
		if err != nil {
			// if the argument is neither a string nor int, error out
			wStrVal, err := getStringArgument(W, w)
			if err != nil {
				return sessionSafety, err
			}
			sessionSafety.WMode = wStrVal
		} else {
			sessionSafety.W = wValue
		}
	}

	return sessionSafety, nil
}

// ParseWriteConcern takes a string and a boolean indicating whether the requested
// write concern is to be used against a replica set. It then converts the write
// concern string argument into an mgo.Safe object which can safely be used to
// set the write concern on a cluster session connection.
func ParseWriteConcern(writeConcern string, isReplicaSet bool) (*mgo.Safe, error) {
	sessionSafety, err := constructWCObject(writeConcern)
	if err != nil {
		return nil, err
	}

	if sessionSafety == nil {
		log.Logf(log.DebugLow, "using unacknowledged write concern")
		return nil, nil
	}

	// for standalone mongods, only a write concern of 0/1 is needed. This update
	// is only here for compatibility with versions of mongod < 2.6
	if !isReplicaSet {
		log.Logf(log.DebugLow, "standalone server: setting write concern %v to 1", W)
		sessionSafety.W = 1
		sessionSafety.WMode = ""
	}

	var writeConcernStr interface{}

	if sessionSafety.WMode != "" {
		writeConcernStr = sessionSafety.WMode
	} else {
		writeConcernStr = sessionSafety.W
	}
	log.Logf(log.DebugLow, "using write concern: %v='%v', %v=%v, %v=%v, %v=%v",
		W, writeConcernStr,
		J, sessionSafety.J,
		FSYNC, sessionSafety.FSync,
		WTIMEOUT, sessionSafety.WTimeout,
	)
	return sessionSafety, nil
}
