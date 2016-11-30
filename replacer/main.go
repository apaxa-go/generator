package main

import "os"
import "github.com/apaxa-go/generator/replacer/internal"

// Sample usage:
// //replacer:ignore
// //go:generate go run $GOPATH/src/github.com/apaxa-go/generator/replacer/main.go -- $GOFILE
// //replacer:replace
// //replacer:old int64	Int64
// //replacer:new int	Int
// //replacer:new int8	Int8
// //replacer:new int16	Int16
// //replacer:new int32	Int32

func main() {
	switch len(os.Args) {
	case 2:
		replacer.Produce(os.Args[1])
	case 3:
		replacer.Produce(os.Args[2])
	default:
		panic("Bad usage. Pass 1 or 2 arguments. The last one should be path to file, estimated arguments will be ignored.")
	}
}
