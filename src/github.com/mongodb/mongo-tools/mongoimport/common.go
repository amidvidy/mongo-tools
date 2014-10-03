package mongoimport

import (
	"fmt"
	"github.com/mongodb/mongo-tools/common/log"
	"gopkg.in/mgo.v2/bson"
	"sort"
	"strconv"
	"strings"
)

// validateHeaders takes an ImportInput, and does some validation on the
// header fields. It returns an error if an issue is found in the header list
func validateHeaders(importInput ImportInput, hasHeaderLine bool) (validatedFields []string, err error) {
	unsortedHeaders := []string{}
	existingImportFields := []string{}

	if reader, isCSV := importInput.(*CSVImportInput); isCSV {
		existingImportFields = reader.Fields
	} else if reader, isTSV := importInput.(*TSVImportInput); isTSV {
		existingImportFields = reader.Fields
	} else {
		return nil, fmt.Errorf("importInput '%#v' unknown", importInput)
	}

	// NOTE: if --headerline was passed on the command line, we will
	// attempt to read headers from the input source - even if --fields
	// or --fieldFile is supplied.
	// TODO: add validation for this case
	if hasHeaderLine {
		if reader, isCSV := importInput.(*CSVImportInput); isCSV {
			unsortedHeaders, err = reader.csvReader.Read()
			if err != nil {
				return nil, err
			}
		} else if reader, isTSV := importInput.(*TSVImportInput); isTSV {
			stringHeaders, err := reader.tsvReader.ReadString(entryDelimiter)
			if err != nil {
				return nil, err
			}
			tokenizedHeaders := strings.Split(stringHeaders, tokenSeparator)
			for _, header := range tokenizedHeaders {
				unsortedHeaders = append(unsortedHeaders, strings.TrimSpace(header))
			}
		}
	} else {
		unsortedHeaders = existingImportFields
	}

	headers := make([]string, len(unsortedHeaders), len(unsortedHeaders))
	copy(headers, unsortedHeaders)
	sort.Sort(sort.StringSlice(headers))

	for index, header := range headers {
		if strings.HasSuffix(header, ".") || strings.HasPrefix(header, ".") {
			return nil, fmt.Errorf("header '%v' can not start or end in '.'", header)
		}
		if strings.Contains(header, "..") {
			return nil, fmt.Errorf("header '%v' can not contain consecutive '.' characters", header)
		}
		// NOTE: since headers is sorted, this check ensures that no header
		// is incompatible with another one that occurs further down the list.
		// meant to prevent cases where we have headers like "a" and "a.c"
		for _, latterHeader := range headers[index+1:] {
			if strings.HasPrefix(latterHeader, header) &&
				(strings.Contains(header, ".") ||
					strings.Contains(latterHeader, ".")) {
				return nil, fmt.Errorf("incompatible headers found: '%v' and '%v",
					header, latterHeader)
			}
			// NOTE: this means we will not support imports that have fields like
			// a,a - since this is invalid in MongoDB
			if header == latterHeader {
				return nil, fmt.Errorf("headers can not be identical: '%v' and '%v",
					header, latterHeader)
			}
		}
		validatedFields = append(validatedFields, unsortedHeaders[index])
	}
	if len(headers) == 1 {
		log.Logf(1, "using field: %v", validatedFields[0])
	} else {
		log.Logf(1, "using fields: %v", strings.Join(validatedFields, ","))
	}
	return validatedFields, nil
}

// getParsedValue returns the appropriate concrete type for the given token
// it first attempts to convert it to an int, if that doesn't succeed, it
// attempts conversion to a float, if that doesn't succeed, it returns the
// token as is.
func getParsedValue(token string) interface{} {
	parsedInt, err := strconv.Atoi(strings.Trim(token, " "))
	if err == nil {
		return parsedInt
	}
	parsedFloat, err := strconv.ParseFloat(strings.Trim(token, " "), 64)
	if err == nil {
		return parsedFloat
	}
	return token
}

// setNestedValue takes a nested field - in the form "a.b.c" -
// its associated value, and a document. It then assigns that
// value to the appropriate nested field within the document
func setNestedValue(field string, value interface{}, document bson.M) {
	index := strings.Index(field, ".")
	if index == -1 {
		document[field] = value
		return
	}
	left := field[0:index]
	subDocument := bson.M{}
	if document[left] != nil {
		subDocument = document[left].(bson.M)
	}
	setNestedValue(field[index+1:], value, subDocument)
	document[left] = subDocument
}
