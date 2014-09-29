KERBEROS_FLAGS=-tags sasl

mongotop-kerberos:
	go build -o mongotop $(KERBEROS_FLAGS) src/github.com/mongodb/mongo-tools/mongotop/main/mongotop.go

mongotop:
	go build -o mongotop src/github.com/mongodb/mongo-tools/mongotop/main/mongotop.go

mongoexport-kerberos:
	go build -o mongoexport $(KERBEROS_FLAGS) src/github.com/mongodb/mongo-tools/mongoexport/main/mongoexport.go
