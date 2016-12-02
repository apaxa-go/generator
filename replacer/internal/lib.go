package replacer

import (
	"github.com/apaxa-go/helper/pathh/filepathh"
	"github.com/apaxa-go/helper/stringsh"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Try to not use of apaxa-go/helper lib to avoid recursive dependencies

const (
	prefix  = "replacer"
	delim   = ":"
	comment = "//"
)

const maxFileSizeForOverwrite = 1024 * 1024 // 1 Mb
const fileSuffix = "-gen"

type parseMode string

// Block types ("// Replacer.<value>")
const (
	ignore    = "ignore"
	replace   = "replace"
	noreplace = "noreplace"
)

// Other possible value ("// Replacer.<value>")
const (
	oldValue  = "old"
	newValue  = "new"
	generated = "generated-file"
)

//type mode string
//
//const (
//	thisFile      mode = "this-file"
//	singleFile         = "single-file"
//	multipleFiles      = "multiple-files"
//)

type block interface {
	Produce() string
}

type asIsBlock struct {
	data string
}

func (b asIsBlock) Produce() string {
	return b.data
}

type replacementBlock struct {
	old  []string   // Map old-value to new-value for replacement
	new  [][]string // len(new) = number of replacement iterations
	data string
}

func (b replacementBlock) Produce() (r string) {
	for i := 0; i < len(b.new); i++ {
		r += stringsh.ReplaceMulti(b.data, b.old, b.new[i])
	}
	return
}

func getBlockData(data string) (estData string, blockData string) {
	const lookFor = comment + prefix + delim
	i := strings.Index(data, lookFor)

	if i == -1 {
		return "", data
	}

	return data[i:], data[:i]
}

// reqName may be empty
func extractDirective(data string, reqName string) (estData, settingsName, settingsValue string, ok bool) {
	const lookFor = comment + prefix + delim

	ok = strings.HasPrefix(data, lookFor+reqName)
	if !ok {
		return data, "", "", false
	}
	if reqName != "" {
		r, _ := utf8.DecodeRuneInString(data[len(lookFor+reqName):]) // Get rune after reqName. It should be space-like or EOF should occurs.
		if !unicode.IsSpace(r) && r != utf8.RuneError {              // RuneError catch EOF case
			return data, "", "", false
		}
	}

	data = data[len(lookFor):]

	tmp, estData := stringsh.ExtractLine(data)

	tmp2 := strings.SplitN(tmp, " ", 2)

	switch len(tmp2) {
	case 2:
		settingsValue = tmp2[1]
		fallthrough
	case 1:
		settingsName = tmp2[0]
	}

	return
}

func splitToBlocks(data string) []block {
	blocks := make([]block, 0, 10)
	origLen := len(data)
	for len(data) > 0 {
		//log.Println(origLen-len(data))
		var name, value string
		var ok bool
		if data, name, value, ok = extractDirective(data, ""); len(name) == 0 || len(value) != 0 || !ok {
			//log.Println(data,name,value,ok)
			panic("Bad line starting from " + strconv.FormatInt(int64(origLen-len(data)), 10) + " byte: '" + stringsh.GetFirstLine(data) + "'")
		}

		//log.Println(data,name,value,ok)

		switch name {
		case ignore:
			data, _ = getBlockData(data)
		case noreplace:
			var b asIsBlock
			data, b.data = getBlockData(data)
			blocks = append(blocks, b)
		case replace:
			var b replacementBlock

			// Read config
			if data, _, value, ok = extractDirective(data, oldValue); !ok {
				panic("Replacement block should have old value in the line after block definition.")
			}
			b.old = strings.Fields(value)
			for data, _, value, ok = extractDirective(data, newValue); ok; data, _, value, ok = extractDirective(data, newValue) {
				tmp := strings.Fields(value)
				if len(tmp) != len(b.old) {
					panic("new values should be exactly the same number as old")
				}
				b.new = append(b.new, tmp)
			}

			//////
			data, b.data = getBlockData(data)
			blocks = append(blocks, b)
		default:
			panic("unknown block type")
		}
	}

	return blocks
}

func produceStr(data string) string {
	var r string
	for _, b := range splitToBlocks(data) {
		r += b.Produce()
	}
	return r
}

func isOverwriteSafe(fn string) bool {
	if stat, err := os.Stat(fn); os.IsNotExist(err) {
		return true
	} else if !stat.Mode().IsRegular() {
		return false
	} else if stat.Size() > maxFileSizeForOverwrite {
		return false
	} else if stat.Size() == 0 {
		return true
	}

	tmp, err := ioutil.ReadFile(fn)
	if err != nil {
		return false
	}
	_, _, _, ok := extractDirective(string(tmp), generated)
	return ok

}

// Produce does all work )
func Produce(fn string) {
	var targetFn string
	{
		base, ext := filepathh.ExtractExt(fn)
		optionalSuffix := "_test"
		if strings.HasSuffix(base, optionalSuffix) {
			base = base[:len(base)-len(optionalSuffix)]
			ext = optionalSuffix + ext
		}
		targetFn = base + fileSuffix + ext

	}
	if !isOverwriteSafe(targetFn) {
		panic("Target file " + targetFn + " : it is not safe to overwrite it")
	}

	var data string
	if tmp, err := ioutil.ReadFile(fn); err != nil {
		panic("Unable to read file " + fn + " : " + err.Error())
	} else {
		data = comment + prefix + delim + noreplace + "\n" + string(tmp)
	}

	data = comment + prefix + delim + generated + "\n\n" + produceStr(data) // Use double new-line to avoid godoc from parsing generated mark

	// Format output
	if fData, err := format.Source([]byte(data)); err == nil {
		data = string(fData)
	} else {
		log.Print("Unable to format result source file: ", err)
	}

	if err := ioutil.WriteFile(targetFn, []byte(data), 0777); err != nil {
		panic("Unable to write " + targetFn + " : " + err.Error())
	}
}
