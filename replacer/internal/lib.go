package replacer

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Do not use apaxa-go/helper lib to avoid recursive dependencies

const (
	prefix   = "replacer"
	delim    = ":"
	comment  = "//"
	valueSep = ","
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
		r += ReplaceMulti(b.data, b.old, b.new[i])
	}
	return
}

func GetBlockData(data string) (estData string, blockData string) {
	const lookFor = comment + prefix + delim
	i := strings.Index(data, lookFor)

	if i == -1 {
		return "", data
	}

	return data[i:], data[:i]
}

// TODO move to other package
func getLine(s string) (string, int) {
	i := strings.Index(s, "\n")
	if i == -1 {
		return s, len(s)
	}

	if i > 0 && s[i-1] == '\r' {
		return s[:i-1], i + 1
	}

	return s[:i], i + 1
}

// TODO move to other package
func GetLine(s string) string {
	line, _ := getLine(s)
	return line
}

// TODO move to other package
func ExtractLine(s string) (line, est string) {
	//log.Println(s)
	line, l := getLine(s)
	if l < len(s) {
		est = s[l:]
	}
	//log.Println(line,est,l)
	return
}

// reqName may be empty
func ExtractDirective(data string, reqName string) (estData, settingsName, settingsValue string, ok bool) {
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

	tmp, estData := ExtractLine(data)

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

func SplitToBlocks(data string) []block {
	blocks := make([]block, 0, 10)
	origLen := len(data)
	for len(data) > 0 {
		//log.Println(origLen-len(data))
		var name, value string
		var ok bool
		if data, name, value, ok = ExtractDirective(data, ""); len(name) == 0 || len(value) != 0 || !ok {
			//log.Println(data,name,value,ok)
			panic("Bad line starting from " + strconv.FormatInt(int64(origLen-len(data)), 10) + " byte: '" + GetLine(data) + "'")
		}

		//log.Println(data,name,value,ok)

		switch name {
		case ignore:
			data, _ = GetBlockData(data)
		case noreplace:
			var b asIsBlock
			data, b.data = GetBlockData(data)
			blocks = append(blocks, b)
		case replace:
			var b replacementBlock

			// Read config
			if data, _, value, ok = ExtractDirective(data, oldValue); !ok {
				panic("Replacement block should have old value in the line after block definition.")
			}
			b.old = strings.Fields(value)
			for data, _, value, ok = ExtractDirective(data, newValue); ok; data, _, value, ok = ExtractDirective(data, newValue) {
				tmp := strings.Fields(value)
				if len(tmp) != len(b.old) {
					panic("new values should be exactly the same number as old")
				}
				b.new = append(b.new, tmp)
			}

			//////
			data, b.data = GetBlockData(data)
			blocks = append(blocks, b)
		default:
			panic("unknown block type")
		}
	}

	return blocks
}

func ProduceStr(data string) string {
	var r string
	for _, b := range SplitToBlocks(data) {
		r += b.Produce()
	}
	return r
}

// TODO move to other package
func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func IsOverwriteSafe(fn string) bool {
	if stat, err := os.Stat(fn); os.IsNotExist(err) {
		return true
	} else if !stat.Mode().IsRegular() {
		return false
	} else if stat.Size() > maxFileSizeForOverwrite {
		return false
	} else if stat.Size() == 0 {
		return true
	}

	if tmp, err := ioutil.ReadFile(fn); err != nil {
		return false
	} else {
		_, _, _, ok := ExtractDirective(string(tmp), generated)
		return ok
	}
}

// TODO move to other package
func SplitExt(path string) (base, ext string) {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[:i], path[i:]
		}
	}
	return path, ""
}

func Produce(fn string) {
	var targetFn string
	{
		base, ext := SplitExt(fn)
		optionalSuffix := "_test"
		if strings.HasSuffix(base, optionalSuffix) {
			base = base[:len(base)-len(optionalSuffix)]
			ext = optionalSuffix + ext
		}
		targetFn = base + fileSuffix + ext

	}
	if !IsOverwriteSafe(targetFn) {
		panic("Target file " + targetFn + " : it is not safe to overwrite it")
	}

	var data string
	if tmp, err := ioutil.ReadFile(fn); err != nil {
		panic("Unable to read file " + fn + " : " + err.Error())
	} else {
		data = comment + prefix + delim + noreplace + "\n" + string(tmp)
	}

	data = comment + prefix + delim + generated + "\n" + ProduceStr(data)

	if err := ioutil.WriteFile(targetFn, []byte(data), 0777); err != nil {
		panic("Unable to write " + targetFn + " : " + err.Error())
	}
}

// TODO move to other package
// Index returns the index of the first instance of any seps in s and founded sep (its index in seps), or (-1,-1) if seps are not present in s.
func IndexMulti(s string, seps []string) (i int, sep int) {
	for i = range s {
		for j := range seps {
			if strings.HasPrefix(s[i:], seps[j]) {
				return i, j
			}
		}
	}

	return -1, -1
}

// TODO move to other package
// Replace returns a copy of the string s with non-overlapping instances of old elements replaced by corresponding new elements.
// If len(old) != len(new) => panic
// if len(old[i])==0 => panic
// If old is empty, Replace return s as-is.
func ReplaceMulti(s string, old, new []string) (r string) {
	if len(old) != len(new) {
		panic("ReplaceMulti: number of old elemnts and new elemnts should be the same.")
	}

	for i := range old {
		if len(old[i]) == 0 {
			panic("ReplaceMulti: no one old can be empty string.")
		}
	}

	if len(old) == 0 {
		return s
	}

	for i, j := IndexMulti(s, old); i != -1; i, j = IndexMulti(s, old) {
		r += s[:i] + new[j]
		s = s[i+len(old[j]):]
	}
	r += s
	return
}
