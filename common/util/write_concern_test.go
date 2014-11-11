package util

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetBoolArgument(t *testing.T) {
	Convey("Given an argument name and an interface value...", t, func() {
		Convey("no error should be returned if the value can be converted "+
			"to a boolean", func() {
			value, err := getBoolArgument("", "true")
			So(err, ShouldBeNil)
			So(value, ShouldBeTrue)
			value, err = getBoolArgument("", "false")
			So(err, ShouldBeNil)
			So(value, ShouldBeFalse)
		})
		Convey("an error should be returned if the value can not be converted "+
			"to a boolean", func() {
			_, err := getBoolArgument("", "truer")
			So(err, ShouldNotBeNil)
			_, err = getBoolArgument("", "3")
			So(err, ShouldNotBeNil)
			_, err = getBoolArgument("", 5)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetStringArgument(t *testing.T) {
	Convey("Given an argument name and an interface value...", t, func() {
		Convey("no error should be returned if the value can be converted "+
			"to a string", func() {
			strVal := "3"
			value, err := getStringArgument("", strVal)
			So(err, ShouldBeNil)
			So(value, ShouldEqual, strVal)
		})
		Convey("an error should be returned if the value can not be converted "+
			"to a string", func() {
			_, err := getStringArgument("", 5)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetIntArgument(t *testing.T) {
	Convey("Given an argument name and an interface value...", t, func() {
		Convey("no error should be returned if the value can be converted "+
			"to a int", func() {
			expectedVal := 3
			value, err := getIntArgument("", "3")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, expectedVal)
			value, err = getIntArgument("", 3)
			So(err, ShouldBeNil)
			So(value, ShouldEqual, expectedVal)
			value, err = getIntArgument("", 3.432)
			So(err, ShouldBeNil)
			So(value, ShouldEqual, expectedVal)
		})
		Convey("an error should be returned if the value can not be converted "+
			"to a int", func() {
			_, err := getIntArgument("", "hello")
			So(err, ShouldNotBeNil)
			_, err = getIntArgument("", "3df")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestParseWriteConcern(t *testing.T) {
	Convey("Given a write concern string value, and a boolean indicating if the "+
		"write concern is to be used on a replica set, on calling ParseWriteConcern...", t, func() {
		Convey("no error should be returned if the write concern is valid", func() {
			writeConcern, err := ParseWriteConcern(`{w:34}`, true)
			So(err, ShouldBeNil)
			So(writeConcern.W, ShouldEqual, 34)
			writeConcern, err = ParseWriteConcern(`{w:"majority"}`, true)
			So(err, ShouldBeNil)
			So(writeConcern.WMode, ShouldEqual, "majority")
			writeConcern, err = ParseWriteConcern(`tagset`, true)
			So(err, ShouldBeNil)
			So(writeConcern.WMode, ShouldEqual, "tagset")
		})
		Convey("on replica sets, only a write concern of 1 or 0 should be returned", func() {
			writeConcern, err := ParseWriteConcern(`{w:34}`, false)
			So(err, ShouldBeNil)
			So(writeConcern.W, ShouldEqual, 1)
			writeConcern, err = ParseWriteConcern(`{w:"majority"}`, false)
			So(err, ShouldBeNil)
			So(writeConcern.W, ShouldEqual, 1)
			writeConcern, err = ParseWriteConcern(`tagset`, false)
			So(err, ShouldBeNil)
			So(writeConcern.W, ShouldEqual, 1)
		})
		Convey("with a w value of 0, without j set, a nil write concern should be returned", func() {
			writeConcern, err := ParseWriteConcern(`{w:0}`, false)
			So(err, ShouldBeNil)
			So(writeConcern, ShouldBeNil)
		})
		Convey("with a w value of 0, with j set, a non-nil write concern should be returned", func() {
			writeConcern, err := ParseWriteConcern(`{w:0, j:true}`, false)
			So(err, ShouldBeNil)
			So(writeConcern.J, ShouldBeTrue)
		})
	})
}

func TestConstructWCObject(t *testing.T) {
	Convey("Given a write concern string value, on calling constructWCObject...", t, func() {

		Convey("non-JSON string values should be assigned to the 'WMode' "+
			"field in their entirety", func() {
			writeConcernString := "majority"
			writeConcern, err := constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.WMode, ShouldEqual, writeConcernString)
		})

		Convey("non-JSON int values should be assigned to the 'w' field "+
			"in their entirety", func() {
			writeConcernString := "43"
			expectedW := 43
			writeConcern, err := constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.W, ShouldEqual, expectedW)
		})

		Convey("JSON strings with valid j, wtimeout, fsync and w, should be "+
			"assigned accordingly", func() {
			writeConcernString := `{w: 3, j: true, fsync: false, wtimeout: 43}`
			expectedW := 3
			expectedWTimeout := 43
			writeConcern, err := constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.W, ShouldEqual, expectedW)
			So(writeConcern.J, ShouldBeTrue)
			So(writeConcern.FSync, ShouldBeFalse)
			So(writeConcern.WTimeout, ShouldEqual, expectedWTimeout)
		})

		Convey("JSON strings with an invalid j argument should error out", func() {
			writeConcernString := `{w: 3, j: "rue"}`
			_, err := constructWCObject(writeConcernString)
			So(err, ShouldNotBeNil)
		})

		Convey("JSON strings with an invalid fsync argument should error out", func() {
			writeConcernString := `{w: 3, fsync: "rue"}`
			_, err := constructWCObject(writeConcernString)
			So(err, ShouldNotBeNil)
		})

		Convey("JSON strings with an invalid wtimeout argument should error out", func() {
			writeConcernString := `{w: 3, wtimeout: "rue"}`
			_, err := constructWCObject(writeConcernString)
			So(err, ShouldNotBeNil)
		})

		Convey("JSON strings with a shorthand j argument should not error out", func() {
			writeConcernString := `{w: 3, j: "t"}`
			writeConcern, err := constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.J, ShouldBeTrue)
			writeConcernString = `{w: 3, j: "f"}`
			writeConcern, err = constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.J, ShouldBeFalse)
		})

		Convey("JSON strings with a shorthand fsync argument should not error out", func() {
			writeConcernString := `{w: 3, fsync: "t"}`
			writeConcern, err := constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.FSync, ShouldBeTrue)
			writeConcernString = `{w: "3", fsync: "f"}`
			writeConcern, err = constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern.FSync, ShouldBeFalse)
		})

		Convey("Unacknowledge write concern strings should return a nil object "+
			"if journaling is not required", func() {
			writeConcernString := `{w: 0}`
			writeConcern, err := constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern, ShouldBeNil)
			writeConcernString = `{w: "0"}`
			writeConcern, err = constructWCObject(writeConcernString)
			So(err, ShouldBeNil)
			So(writeConcern, ShouldBeNil)
		})
	})
}
