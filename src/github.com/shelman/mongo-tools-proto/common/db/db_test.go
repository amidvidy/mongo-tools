package db

import (
	"github.com/shelman/mongo-tools-proto/common/options"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInitSessionProvider(t *testing.T) {

	Convey("When initializing a session provider", t, func() {

		Convey("the host and port specified should propagate to the"+
			" provider's dial info", func() {

			opts := options.New("", "", "")
			opts.Connection = &options.Connection{
				Host: "host1,host2",
				Port: "20000",
			}
			provider, err := InitSessionProvider(opts)
			So(err, ShouldBeNil)
			So(provider.dialInfo.Addrs, ShouldResemble, []string{"host1:20000",
				"host2:20000"})

		})

		Convey("if ssl is not specified, no dialer function should be"+
			" created", func() {

			opts := options.New("", "", "")
			provider, err := InitSessionProvider(opts)
			So(err, ShouldBeNil)
			So(provider.dialInfo.DialServer, ShouldBeNil)

		})

		Convey("if ssl is specified, a dialer function to use ssl should"+
			" be created and used by the provider", func() {

			opts := options.New("", "", "")
			opts.SSL = &options.SSL{
				UseSSL: true,
			}
			provider, err := InitSessionProvider(opts)
			So(err, ShouldBeNil)
			So(provider.dialInfo.DialServer, ShouldNotBeNil)

		})

	})

}

func TestCreateConnectionAddrs(t *testing.T) {

	Convey("When creating the slice of connection addresses", t, func() {

		Convey("if no port is specified, the addresses should all appear"+
			" unmodified in the result", func() {

			addrs := createConnectionAddrs("host1,host2", "")
			So(addrs, ShouldResemble, []string{"host1", "host2"})

		})

		Convey("if a port is specified, it should be appended to each host"+
			" from the host connection string", func() {

			addrs := createConnectionAddrs("host1,host2", "20000")
			So(addrs, ShouldResemble, []string{"host1:20000", "host2:20000"})

		})

	})

}

func TestParseHost(t *testing.T) {

	Convey("When parsing a host string into the contained"+
		" addresses", t, func() {

		Convey("a string with a single hostname should return a slice of just"+
			" the hostname", func() {

			addrs := parseHost("localhost")
			So(addrs, ShouldResemble, []string{"localhost"})

		})

		Convey("a string with multiple hostnames should return a slice of"+
			" all of them", func() {

			addrs := parseHost("host1,host2,host3")
			So(addrs, ShouldResemble, []string{"host1", "host2", "host3"})

		})

		Convey("a string with multiple hostnames and a replica set should"+
			" return a slice of all the host names", func() {

			addrs := parseHost("foo/host1,host2,host3")
			So(addrs, ShouldResemble, []string{"host1", "host2", "host3"})

		})

	})
}
