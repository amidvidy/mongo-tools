package mongoimport

import (
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/mgo.v2/bson"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestCSVImportDocument(t *testing.T) {
	Convey("With a CSV import input", t, func() {
		var err error
		var csvFile, fileHandle *os.File
		Convey("badly encoded csv should result in a parsing error", func() {
			contents := `1, 2, foo"bar`
			fields := []string{"a", "b", "c"}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err := os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			_, err = csvImporter.ImportDocument()
			So(err, ShouldNotBeNil)
		})
		Convey("escaped quotes are parsed correctly", func() {
			contents := `1, 2, "foo""bar"`
			fields := []string{"a", "b", "c"}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err := os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			_, err = csvImporter.ImportDocument()
			So(err, ShouldBeNil)
		})
		Convey("whitespace separated quoted strings are still an error", func() {
			contents := `1, 2, "foo"  "bar"`
			fields := []string{"a", "b", "c"}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err := os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			_, err = csvImporter.ImportDocument()
			So(err, ShouldNotBeNil)
		})
		Convey("multiple escaped quotes separated by whitespace parsed correctly", func() {
			contents := `1, 2, "foo"" ""bar"`
			fields := []string{"a", "b", "c"}
			expectedRead := bson.M{
				"a": 1,
				"b": 2,
				"c": `foo" "bar`,
			}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err := os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			bsonDoc, err := csvImporter.ImportDocument()
			So(err, ShouldBeNil)
			So(bsonDoc, ShouldResemble, expectedRead)
		})
		Convey("integer valued strings should be converted", func() {
			contents := `1, 2, " 3e"`
			fields := []string{"a", "b", "c"}
			expectedRead := bson.M{
				"a": 1,
				"b": 2,
				"c": " 3e",
			}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err := os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			bsonDoc, err := csvImporter.ImportDocument()
			So(err, ShouldBeNil)
			So(bsonDoc, ShouldResemble, expectedRead)
		})

		Convey("extra fields should be prefixed with 'field'", func() {
			contents := `1, 2f , " 3e" , " may"`
			fields := []string{"a", "b", "c"}
			expectedRead := bson.M{
				"a":      1,
				"b":      "2f",
				"c":      " 3e",
				"field3": " may",
			}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			bsonDoc, err := csvImporter.ImportDocument()
			So(err, ShouldBeNil)
			So(bsonDoc, ShouldResemble, expectedRead)
		})

		Convey("nested csv fields should be imported properly", func() {
			contents := `1, 2f , " 3e" , " may"`
			fields := []string{"a", "b.c", "c"}
			expectedRead := bson.M{
				"a": 1,
				"b": bson.M{
					"c": "2f",
				},
				"c":      " 3e",
				"field3": " may",
			}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			bsonDoc, err := csvImporter.ImportDocument()
			So(err, ShouldBeNil)
			So(bsonDoc, ShouldResemble, expectedRead)
		})

		Convey("nested csv fields causing header collisions should error", func() {
			contents := `1, 2f , " 3e" , " may", june`
			fields := []string{"a", "b.c", "field3"}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			_, err := csvImporter.ImportDocument()
			So(err, ShouldNotBeNil)
		})

		Convey("calling ImportDocument() for CSVs should return next set of "+
			"values", func() {
			contents := "1, 2, 3\n4, 5, 6"
			fields := []string{"a", "b", "c"}
			expectedReadOne := bson.M{
				"a": 1,
				"b": 2,
				"c": 3,
			}
			expectedReadTwo := bson.M{
				"a": 4,
				"b": 5,
				"c": 6,
			}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err := os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			bsonDoc, err := csvImporter.ImportDocument()
			So(err, ShouldBeNil)
			So(bsonDoc, ShouldResemble, expectedReadOne)
			bsonDoc, err = csvImporter.ImportDocument()
			So(err, ShouldBeNil)
			So(bsonDoc, ShouldResemble, expectedReadTwo)
		})

		Reset(func() {
			csvFile.Close()
			fileHandle.Close()
		})
	})
}

func TestCSVSetHeader(t *testing.T) {
	var err error
	var csvFile, fileHandle *os.File
	Convey("With a CSV import input", t, func() {
		Convey("setting the header should read the first line of the CSV",
			func() {
				contents := "extraHeader1, extraHeader2, extraHeader3"
				fields := []string{}

				csvFile, err = ioutil.TempFile("", "mongoimport_")
				So(err, ShouldBeNil)
				_, err = io.WriteString(csvFile, contents)
				So(err, ShouldBeNil)
				fileHandle, err = os.Open(csvFile.Name())
				So(err, ShouldBeNil)
				csvImporter := NewCSVImportInput(fields, fileHandle)
				So(csvImporter.SetHeader(), ShouldBeNil)
				So(len(csvImporter.Fields), ShouldEqual, 3)
			})

		Convey("setting non-colliding nested csv headers should not raise an error", func() {
			contents := "a, b, c"
			fields := []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldBeNil)
			So(len(csvImporter.Fields), ShouldEqual, 3)

			contents = "a.b.c, a.b.d, c"
			fields = []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter = NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldBeNil)
			So(len(csvImporter.Fields), ShouldEqual, 3)

			contents = "a.b, ab, a.c"
			fields = []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter = NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldBeNil)
			So(len(csvImporter.Fields), ShouldEqual, 3)

			contents = "a, ab, ac, dd"
			fields = []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter = NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldBeNil)
			So(len(csvImporter.Fields), ShouldEqual, 4)
		})

		Convey("setting colliding nested csv headers should raise an error", func() {
			contents := "a, a.b, c"
			fields := []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter := NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldNotBeNil)

			contents = "a.b.c, a.b.d.c, a.b.d"
			fields = []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter = NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldNotBeNil)

			contents = "a, a, a"
			fields = []string{}

			csvFile, err = ioutil.TempFile("", "mongoimport_")
			So(err, ShouldBeNil)
			_, err = io.WriteString(csvFile, contents)
			So(err, ShouldBeNil)
			fileHandle, err = os.Open(csvFile.Name())
			So(err, ShouldBeNil)
			csvImporter = NewCSVImportInput(fields, fileHandle)
			So(csvImporter.SetHeader(), ShouldNotBeNil)
		})

		Convey("setting the header using an empty file should return EOF",
			func() {
				contents := ""
				fields := []string{}

				csvFile, err = ioutil.TempFile("", "mongoimport_")
				So(err, ShouldBeNil)
				_, err = io.WriteString(csvFile, contents)
				So(err, ShouldBeNil)
				fileHandle, err = os.Open(csvFile.Name())
				So(err, ShouldBeNil)
				csvImporter := NewCSVImportInput(fields, fileHandle)
				So(csvImporter.SetHeader(), ShouldEqual, io.EOF)
				So(len(csvImporter.Fields), ShouldEqual, 0)
			})
		Convey("setting the header with fields already set, should "+
			"the header line with the existing fields",
			func() {
				contents := "extraHeader1, extraHeader2, extraHeader3\n\n"
				fields := []string{"a", "b", "c"}

				csvFile, err = ioutil.TempFile("", "mongoimport_")
				So(err, ShouldBeNil)
				_, err = io.WriteString(csvFile, contents)
				So(err, ShouldBeNil)
				fileHandle, err = os.Open(csvFile.Name())
				So(err, ShouldBeNil)
				csvImporter := NewCSVImportInput(fields, fileHandle)
				So(csvImporter.SetHeader(), ShouldBeNil)
				// if SetHeader() with fields already passed in, the header
				// should be a union of both the fields and the header line
				So(len(csvImporter.Fields), ShouldEqual, 6)
			})

		Convey("plain CSV input file sources should be parsed correctly and "+
			"subsequent imports should parse correctly",
			func() {
				fields := []string{"a", "b", "c"}
				expectedReadOne := bson.M{"a": 1, "b": 2, "c": 3}
				expectedReadTwo := bson.M{"a": 3, "b": 5.4, "c": "string"}
				fileHandle, err := os.Open("testdata/test.csv")
				So(err, ShouldBeNil)
				csvImporter := NewCSVImportInput(fields, fileHandle)
				bsonDoc, err := csvImporter.ImportDocument()
				So(err, ShouldBeNil)
				So(bsonDoc, ShouldResemble, expectedReadOne)
				bsonDoc, err = csvImporter.ImportDocument()
				So(err, ShouldBeNil)
				So(bsonDoc, ShouldResemble, expectedReadTwo)
			})
		Reset(func() {
			csvFile.Close()
			fileHandle.Close()
		})
	})
}

func TestGetParsedValue(t *testing.T) {
	Convey("Given a string token to parse", t, func() {
		Convey("an int token should return the underlying int value",
			func() {
				So(getParsedValue("3"), ShouldEqual, 3)
			})
		Convey("a float token should return the underlying float value",
			func() {
				So(getParsedValue(".33"), ShouldEqual, 0.33)
			})
		Convey("a string token should return the underlying string value",
			func() {
				So(getParsedValue("sd"), ShouldEqual, "sd")
			})
	})
}
