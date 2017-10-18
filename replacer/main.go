// Replacer is a GoLang specific variant of 'sed'.
// It allow to generate GoLang source files from other source files by replacing some text with another.
// Replacer is not as much flexible as original sed but have some language specifics.
// Technically Replacer does not require operated files to be a GoLang source files.
//
// Replacer usage is "replacer [--] <source-file-name> [<target-file-name>]". Source file will be split into blocks, each block will be proceed, result of proceed will be concatenated and saved to target file.
// There are 3 types of blocks: "ignore", "replace" and "noreplace". Each block start from some line and end with some line (not from/to middle of line).
// Block starts from line after the line(s) with block directive and ends at next block directives or at EOF.
// Block directive if string of form "//replacer:<block-type>" (example "//replacer:noreplace").
//
// "ignore" block just skipped (proceed of "ignore" block is empty string).
//
// "noreplace" block return as-is by proceed.
//
// "replace" block is the most interesting types of blocks. It requires additional directives right after the block directive to operate. Additional directives define what to replace and with what replace.
// First additional directive should be "old" directive: "//replacer:old <space-separated-list-of-what-to-replace>".
// After "old" directive should be one or more "new" directives: "//replacer:new <space-separated-list-of-with-what-replace>".
// Number of elements in each "old" and "new" in same block must be equal.
// On block produce each non overlapping instance of "old" elements in block will be replaced with the corresponding elements of the first "new" directive.
// After that the same will be happened with second "new" directive and so on.
// Produce of "replace" block is concatenation of all n replacement (where n - is number of "new" directives).
//
// Lines in original file before first block directive treats as belong to "noreplace" block (as if line "//replcaer:noreplace" will be prepend original file).
//
// Result of file proceed saved to result file.
// If target file name is not specified, then it computed in the following way:
// if base name (name without extensions) ended with "_test" then result name will be "<base-name-without-_test>-gen_test<original-extension>",
// otherwise result name will be "<base-name>-gen<original-extension>".
// Example:
// 	math.go => math-gen.go
// 	math_test.go => math-gen_test.go
//
// Because silently overwriting files is a dangerous operation some kind of safety is implemented. All produced files prepend with special line directive "//replacer:generated-file".
// Replacer will panic if it tries to overwrite file without this directive.
// Also additional empty line added after described directive. This for avoiding show this directive in godoc and does not checked on overwrite by Replacer.
//
// Before saving generated source to file it will be formatted via standard GoLang method. If formatting return error then source will be saved as-is.
//
// Remember which files need to apply Replacer is annoying.
// Replacer can be simply used with go:generate -just add this line in base file "//go:generate go run $GOPATH/src/github.com/apaxa-go/generator/replacer/main.go -- $GOFILE".
// Usually it is not required to apply Replacer to files it produce so add line "//replacer:ignore" before go:generate directive.
// As the result it is possible to just run "go generate ./..." to apply Replacer to all required files.
//
// Note: "go run" does not support "--" as delimiter between its own arguments and arguments to program it runs, but without "--" "go run" treats $GOFILE as part of binary to run.
// To avoid this Replacer accept two form usage: "replacer <source-file-name> [<target-file-name>]" (for manual usage) and "replacer -- <source-file-name> [<target-file-name>]" (for using with "go run").
//
// Quick guide
//
// Get Replacer:
// 	go get github.com/apaxa-go/generator/replacer
//
// Create source file "./math/math.h" with the following content:
// 	package math
//
// 	//replacer:ignore
// 	//go:generate go run $GOPATH/src/github.com/apaxa-go/generator/replacer/main.go -- $GOFILE
// 	//replacer:replace
// 	//replacer:old uint64	Uint64
// 	//replacer:new uint	Uint
// 	//replacer:new uint8	Uint8
// 	//replacer:new uint16	Uint16
// 	//replacer:new uint32	Uint32
//
// 	func MinUint64(a, b uint64) uint64 {
// 		if a <= b {
// 			return a
// 		}
// 		return b
// 	}
//
//
// Apply replacer:
// 	go generate ./math/...
//
// Result file will be "./math/math-gen.go" with the following content:
// 	//replacer:generated-file
//
// 	package math
//
// 	func MinUint(a, b uint) uint {
// 		if a <= b {
// 			return a
// 		}
// 		return b
// 	}
//
// 	func MinUint8(a, b uint8) uint8 {
// 		if a <= b {
// 			return a
// 		}
// 		return b
// 	}
//
// 	func MinUint16(a, b uint16) uint16 {
// 		if a <= b {
// 			return a
// 		}
// 		return b
// 	}
//
// 	func MinUint32(a, b uint32) uint32 {
// 		if a <= b {
// 			return a
// 		}
// 		return b
// 	}
//
//
// For more examples see https://github.com/apaxa-go/helper/tree/master/mathh and https://github.com/apaxa-go/helper/tree/master/strconvh .
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
	const usage = "Bad usage. Usage: \"replacer [--] source-file-name [target-file-name]\""
	const fakeArg = "--"

	l := len(os.Args)
	if l < 2 {
		panic(usage)
	}

	var sourceFn, targetFn string
	{
		argShift := 1 // os.Args[0] is always skipped as it has special meaning.
		if os.Args[1] == fakeArg {
			argShift++ // Skip first argument if it is "--".
		}
		switch l { // We need at least 1 effective argument (source file name), but no more that 2 (second is target file name and is optional).
		case argShift + 1:
			sourceFn = os.Args[argShift]
		case argShift + 2:
			sourceFn = os.Args[argShift]
			targetFn = os.Args[argShift+1]
		default:
			panic(usage)
		}
	}

	replacer.Produce(sourceFn, targetFn)
}
