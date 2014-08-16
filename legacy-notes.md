On legacy tools, if the "-d" option is blank it assumes the default database ('test'). It might be preferable and of lower risk to the user to require that -d is specified, instead of assuming a default value.

#####note: This behavior differs from tool to tool. In mongodump, no -d field means *all* databases instead of "test"

##Mongoimport:
If you supply both --headerline and --fields
(or --fieldFile) it, firstly allows this, and just appends the header
lines to the fields already specified.
This seems confusing at best and could be buggy but alas, the same
behavior is replicated herein to maintain the API contract.
2. If you run:

mongoimport -d test    -c test  --file inputFile  --headerline --stopOnError

where inputFile contains:

{"_id":2}
{"_id":3}
{"_id":2}

mongoimport displays:

connected to: 127.0.0.1
2014-07-25T12:47:44.075-0400 dropping: test.test
2014-07-25T12:47:44.078-0400 imported 3 objects

but in the database:

test> db.test.find()
{
  "_id": 3
}
{
  "_id": 2
}
Fetched 2 record(s) in 1ms -- Index[none]

--headerline should have no effect on JSON input sources


##Mongoexport:

What should be the behavior if the user specifiees both --fields *AND* --fieldFile?
possible behaviors:
   * throw an error and tell the user that only one may be specified.
   * use the union of both settings.
   * always just refer to one and ignore the other

##JSON:
The docs on extended json (see http://docs.mongodb.org/manual/reference/mongodb-extended-json/#date) say that dates should be serialized as {$date: <date>}, such that "<date> is the JSON representation of a 64-bit signed integer for milliseconds since epoch UTC".

But legacy mongoexport exports it as a string, like `{ "$date" : "2014-07-03T09:35:07.422-0400" }`. 
It's unclear if the docs are wrong, mongoexport's implementation is wrong, or both.  Also unclear which would be the "correct" formats for mongoimport to accept.

Legacy mongoimport also accepts some strange/unsupported types, like `NumberLong(...)` and `Date(...)`
To make the new mongoimport replicate the old behavior exactly, we would need to extend the JSON parser to handle these. However, this behavior seems odd because mongoexport does not produce these types, and the formats supported by mongoimport do not match those supported by the shell.

According to the docs, it suggests that the legacy mongoimport has a limit of 16MB for the size of a json array that can be used when importing with --jsonArray. The docs are incorrect, the limit is simply that no single document in the array can exceed the 16MB limit.
