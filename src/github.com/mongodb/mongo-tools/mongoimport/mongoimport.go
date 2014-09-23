package mongoimport

import (
	"errors"
	"fmt"
	"github.com/mongodb/mongo-tools/common/db"
	commonOpts "github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/common/util"
	"github.com/mongodb/mongo-tools/mongoimport/options"
	"gopkg.in/mgo.v2/bson"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	CSV  = "csv"
	TSV  = "tsv"
	JSON = "json"
)

// compile-time interface sanity check
var (
	_ ImportInput = (*CSVImportInput)(nil)
	_ ImportInput = (*TSVImportInput)(nil)
	_ ImportInput = (*JSONImportInput)(nil)
)

var (
	errNsNotFound = errors.New("ns not found")
)

// Wrapper for MongoImport functionality
type MongoImport struct {
	// generic mongo tool options
	ToolOptions *commonOpts.ToolOptions

	// InputOptions defines options used to read data to be ingested
	InputOptions *options.InputOptions

	// IngestOptions defines options used to ingest data into MongoDB
	IngestOptions *options.IngestOptions

	// SessionProvider is used for connecting to the database
	SessionProvider *db.SessionProvider
}

// ImportInput is an interface that specifies how an input source should be
// converted to BSON
type ImportInput interface {
	// ImportDocument reads the given record from the given io.Reader according
	// to the format supported by the underlying ImportInput implementation.
	ImportDocument() (bson.M, error)

	// SetHeader sets the header for the CSV/TSV import when --headerline is
	// specified
	SetHeader() error
}

func (mongoImport *MongoImport) getImportWriter() ImportWriter {
	var upsertFields []string
	if mongoImport.IngestOptions.Upsert &&
		len(mongoImport.IngestOptions.UpsertFields) != 0 {
		upsertFields = strings.Split(mongoImport.IngestOptions.UpsertFields, ",")
	}
	if mongoImport.ToolOptions.DBPath == "" {
		return &DriverImportWriter{
			upsertMode:      mongoImport.IngestOptions.Upsert,
			upsertFields:    upsertFields,
			sessionProvider: mongoImport.SessionProvider,
			session:         nil,
		}
	} else {
		if mongoImport.IngestOptions.Upsert {
			panic("not implemented! see SERVER-15309")
		}
		return &ShimImportWriter{
			upsertMode:   mongoImport.IngestOptions.Upsert,
			upsertFields: upsertFields,
			dbPath:       mongoImport.ToolOptions.DBPath,
			dbName:       mongoImport.ToolOptions.Namespace.DB,
			collection:   mongoImport.ToolOptions.Namespace.Collection,
		}
	}
	//return DriverImportWriter{}
}

// ValidateSettings ensures that the tool specific options supplied for
// MongoImport are valid
func (mongoImport *MongoImport) ValidateSettings() error {
	// Namespace must have a valid database
	if mongoImport.ToolOptions.Namespace.DB == "" {
		return fmt.Errorf("must specify a database")
	}

	// use JSON as default input type
	if mongoImport.InputOptions.Type == "" {
		mongoImport.InputOptions.Type = JSON
	} else {
		if !(mongoImport.InputOptions.Type == TSV ||
			mongoImport.InputOptions.Type == JSON ||
			mongoImport.InputOptions.Type == CSV) {
			return fmt.Errorf("don't know what type [\"%v\"] is",
				mongoImport.InputOptions.Type)
		}
	}

	// ensure headers are supplied for CSV/TSV
	if mongoImport.InputOptions.Type == CSV ||
		mongoImport.InputOptions.Type == TSV {
		if !mongoImport.InputOptions.HeaderLine {
			if mongoImport.InputOptions.Fields == "" &&
				mongoImport.InputOptions.FieldFile == "" {
				return fmt.Errorf("You need to specify fields or have a " +
					"header line to import this file type")
			}
		}
	}

	// ensure we have a valid string to use for the collection
	if mongoImport.ToolOptions.Namespace.Collection == "" {
		if mongoImport.InputOptions.File == "" {
			return fmt.Errorf("must specify a collection or filename")
		}
		fileBaseName := filepath.Base(mongoImport.InputOptions.File)
		lastDotIndex := strings.LastIndex(fileBaseName, ".")
		if lastDotIndex != -1 {
			fileBaseName = fileBaseName[0:lastDotIndex]
		}
		mongoImport.ToolOptions.Namespace.Collection = fileBaseName
		util.PrintlnTimeStamped("no collection specified!")
		util.PrintfTimeStamped("using filename '%v' as collection\n", fileBaseName)
	}
	return nil
}

// getInputReader returns an io.Reader corresponding to the input location
func (mongoImport *MongoImport) getInputReader() (io.ReadCloser, error) {
	if mongoImport.InputOptions.File != "" {
		fmt.Println("it's a file")
		file, err := os.Open(mongoImport.InputOptions.File)
		if err != nil {
			return nil, err
		}
		return file, err
	}
	return os.Stdin, nil
}

// ImportDocuments is used to write input data to the database. It returns the
// number of documents successfully imported to the appropriate namespace and
// any error encountered in doing this
func (mongoImport *MongoImport) ImportDocuments() (int64, error) {
	in, err := mongoImport.getInputReader()
	if err != nil {
		return 0, err
	}

	defer in.Close()

	importInput, err := mongoImport.getImportInput(in)
	if err != nil {
		return 0, err
	}

	if mongoImport.InputOptions.HeaderLine {
		err = importInput.SetHeader()
		if err != nil {
			return 0, err
		}
	}
	return mongoImport.importDocuments(importInput)
}

// importDocuments is a helper to ImportDocuments and does all the ingestion
// work by taking data from the 'importInput' source and writing it to the
// appropriate namespace
func (mongoImport *MongoImport) importDocuments(importInput ImportInput) (docsCount int64, err error) {
	importWriter := mongoImport.getImportWriter()
	connUrl := mongoImport.ToolOptions.Host
	if mongoImport.ToolOptions.Port != "" {
		connUrl = connUrl + ":" + mongoImport.ToolOptions.Port
	}
	fmt.Fprintf(os.Stdout, "connected to: %v\n", connUrl)
	err = importWriter.Open(mongoImport.ToolOptions.Namespace.DB, mongoImport.ToolOptions.Namespace.Collection)
	if err != nil {
		return
	}

	defer func() {
		fmt.Println("closing!")
		err2 := importWriter.Close()
		if err == nil {
			err = err2
		}
	}()

	// drop the database if necessary
	if mongoImport.IngestOptions.Drop {
		util.PrintfTimeStamped("dropping: %v.%v\n", mongoImport.ToolOptions.DB,
			mongoImport.ToolOptions.Collection)

		if err := importWriter.Drop(); err != nil &&
			err.Error() != errNsNotFound.Error() {
			return 0, err
		}
	}

	for {
		document, err := importInput.ImportDocument()
		if err != nil {
			if err == io.EOF {
				return docsCount, nil
			}
			if mongoImport.IngestOptions.StopOnError {
				return docsCount, err
			}
			if document == nil {
				return docsCount, err
			}
			continue
		}

		// ignore blank fields if specified
		if mongoImport.IngestOptions.IgnoreBlanks &&
			mongoImport.InputOptions.Type != JSON {
			document = removeBlankFields(document)
		}
		err = importWriter.Import(document)
		if err != nil {
			if mongoImport.IngestOptions.StopOnError {
				return docsCount, err
			}
			fmt.Fprintf(os.Stderr, "error inserting document: %v\n", err)
			continue
		}
		docsCount++
	}
	return docsCount, nil
}

// removeBlankFields removes empty/blank fields in csv and tsv
func removeBlankFields(document bson.M) bson.M {
	for key, value := range document {
		if reflect.TypeOf(value).Kind() == reflect.String &&
			value.(string) == "" {
			delete(document, key)
		}
	}
	return document
}

// getImportInput returns an implementation of ImportInput which can handle
// transforming tsv, csv, or JSON into appropriate BSON documents
func (mongoImport *MongoImport) getImportInput(in io.Reader) (ImportInput,
	error) {
	var fields []string
	var err error
	// there should be some sanity checks done for field names - e.g. that they
	// don't contain dots
	if len(mongoImport.InputOptions.Fields) != 0 {
		fields = strings.Split(strings.Trim(mongoImport.InputOptions.Fields,
			" "), ",")
	} else if mongoImport.InputOptions.FieldFile != "" {
		fields, err = util.GetFieldsFromFile(mongoImport.InputOptions.FieldFile)
		if err != nil {
			return nil, err
		}
	}
	if mongoImport.InputOptions.Type == CSV {
		return NewCSVImportInput(fields, in), nil
	} else if mongoImport.InputOptions.Type == TSV {
		return NewTSVImportInput(fields, in), nil
	}
	return NewJSONImportInput(mongoImport.InputOptions.JSONArray, in), nil
}
