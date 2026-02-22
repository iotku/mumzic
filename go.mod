module github.com/iotku/mumzic

go 1.20

require (
	github.com/dhowden/tag v0.0.0-20240417053706-3d75831295e8
	github.com/mattn/go-sqlite3 v1.14.34
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	layeh.com/gumble v0.0.0-20200818122324-146f9205029b
)

require (
	github.com/golang/protobuf v1.3.1 // indirect
	gopkg.in/hraban/opus.v2 v2.0.0-20230925203106-0188a62cb302 // indirect
)

replace layeh.com/gumble => github.com/iotku/gumble v0.0.2
