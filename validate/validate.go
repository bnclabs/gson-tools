package main

import "flag"
import "strings"
import "sync"
import "math/rand"
import "runtime/debug"
import "os"
import "fmt"
import "log"
import "io/ioutil"
import "path"
import "reflect"
import "runtime"
import "unsafe"
import "bytes"
import "errors"
import "sort"

import "github.com/prataprc/gson"
import "github.com/prataprc/goparsec"
import "github.com/prataprc/monster"
import mcommon "github.com/prataprc/monster/common"

var _ = fmt.Sprintf("dummy")

var options struct {
	seed    int
	count   int
	input   string
	stop    bool
	par     int
	genout  string
	verbose bool
	debug   bool
	outfd   *os.File
}

func argParse() []string {
	flag.IntVar(&options.seed, "seed", 0,
		"random seed to monster")
	flag.IntVar(&options.count, "count", 1,
		"number of validations")
	flag.StringVar(&options.input, "input", "",
		"validate the supplite json string")
	flag.BoolVar(&options.stop, "stop", false,
		"continue after error")
	flag.IntVar(&options.par, "par", 1,
		"number of parallel routines, applicable only with random validation")
	flag.BoolVar(&options.verbose, "v", false,
		"log in verbose mode")
	flag.BoolVar(&options.debug, "g", false,
		"log in debug mode")
	flag.StringVar(&options.genout, "genout", "",
		"store generated JSON samples into file.")
	flag.Parse()

	if options.seed == 0 {
		options.seed = rand.Int()
	}
	return flag.Args()
}

var statrw sync.RWMutex
var statistics = map[string]interface{}{
	"docs":              0,
	"pass":              0,
	"fail":              0,
	"bytes":             0,
	"null":              0,
	"true":              0,
	"false":             0,
	"num":               0,
	"string":            0,
	"array":             0,
	"object":            0,
	"SmartNumber":       0,
	"FloatNumber":       0,
	"Decimal":           0,
	"AnsiSpace":         0,
	"UnicodeSpace":      0,
	"LengthPrefix":      0,
	"Stream":            0,
	"arrayLenPrefix":    0,
	"propertyLenPrefix": 0,
	"doMissing":         0,
	"strict":            0,
}

func main() {
	argParse()

	var err error
	if options.genout != "" {
		if options.outfd, err = os.Create(options.genout); err != nil {
			log.Fatal(err)
		}
		defer options.outfd.Close()
	}

	defer func() {
		printStatistics()
		if statistics["fail"].(int) > 0 {
			os.Exit(1)
		}
	}()

	if options.input != "" {
		mrand := rand.New(rand.NewSource(int64(options.seed)))
		for i := 0; i < options.count; i++ {
			validateString(mrand, options.input)
		}
	} else {
		validateRandom()
	}
}

func validateRandom() (status map[string]interface{}) {
	_, filename, _, _ := runtime.Caller(0)
	prodfile := path.Join(path.Dir(filename), "2i.json.prod")
	var wg sync.WaitGroup
	wg.Add(options.par)
	ch := generateJSON(prodfile, options.seed, options.count)

	donech := make(chan bool, 1000)
	go func() {
		i := 1
		for range donech {
			if i%10000 == 0 {
				fmt.Printf("completed %v\n", i)
			}
			i++
		}
	}()

	for n := 0; n < options.par; n++ {
		go func(n int) {
			mrand := rand.New(rand.NewSource(int64(options.seed)))
			for data := range ch {
				verbosef(fmt.Sprintf("json: %v\n", data))
				if err := validateString(mrand, data); err != nil {
					if options.stop {
						os.Exit(1)
					}
				}
				donech <- true
			}
			wg.Done()
		}(n)
	}
	wg.Wait()
	return
}

func validateString(mrand *rand.Rand, jsonstr string) (err error) {
	config := makeConfig(mrand)
	data := str2bytes(jsonstr)
	jsn := config.NewJson(str2bytes(jsonstr), -1)

	_, doc := jsn.Tovalue()

	defer func() { bookstats(config, jsonstr, doc, err) }()

	// validate pointer ops
	switch doc := doc.(type) {
	case []interface{}:
		if err = verifyValuePointers(config, doc); err != nil {
			return
		}
		if err = verifyCborPointers(config, doc); err != nil {
			fmsg := "fail verifyCborPointers: %v\njson: %v\n\n"
			printFailure(config, fmsg, err, jsonstr)
			return
		}

	case map[string]interface{}:
		if err = verifyValuePointers(config, doc); err != nil {
			return
		}
		if err = verifyCborPointers(config, doc); err != nil {
			fmsg := "fail verifyCborPointers: %v\njson: %v\n\n"
			printFailure(config, fmsg, err, jsonstr)
			return
		}
	}
	// validate transforms
	if err = value2json2cbor2collate(config, data); err != nil {
		fmsg := "fail value2json2cbor2collate: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = value2cbor2collate(config, data); err != nil {
		fmsg := "fail value2cbor2collate: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = value2collate(config, data); err != nil {
		fmsg := "fail value2collate: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = json2cbor2collate2value(config, data); err != nil {
		fmsg := "fail json2cbor2collate2value: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = json2collate2value(config, data); err != nil {
		fmsg := "fail json2collate2value: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = json2value(config, data); err != nil {
		fmsg := "fail json2value: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = cbor2collate2value2json(config, data); err != nil {
		fmsg := "fail cbor2collate2value2json: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = cbor2value2json(config, data); err != nil {
		fmsg := "fail cbor2value2json: %v\njson:%v \n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = cbor2json(config, data); err != nil {
		fmsg := "fail cbor2json: %v\njson:%v \n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = collate2value2json2cbor(config, data); err != nil {
		fmsg := "fail collate2value2json2cbor: %v\njson:%v \n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = collate2json2cbor(config, data); err != nil {
		fmsg := "fail collate2json2cbor: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	if err = collate2cbor(config, data); err != nil {
		fmsg := "fail collate2cbor: %v\njson: %v\n\n"
		printFailure(config, fmsg, err, jsonstr)
		return
	}
	return
}

func value2json2cbor2collate(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// value -> json -> cbor -> collate -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	rval := config.NewValue(ref)
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	clt := config.NewCollate(make([]byte, 1024), 0)

	value := rval.Tojson(jsn).Tocbor(cbr).Tocollate(clt).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		x, y := ref.([]interface{}), value.([]interface{})
		//return fmt.Errorf("DeepEqual() expected %T, got %T", ref, value)
		return fmt.Errorf("DeepEqual() expected %T, got %T", x[3], y[3])
	}
	verbosef("value2json2cbor2collate ... ok\n")
	return
}

func value2cbor2collate(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> value -> cbor -> collate -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	rval := config.NewValue(ref)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	clt := config.NewCollate(make([]byte, 1024), 0)

	value := rval.Tocbor(cbr).Tocollate(clt).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("value2cbor2collate ... ok\n")
	return
}

func value2collate(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> value -> collate -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	rval := config.NewValue(ref)
	clt := config.NewCollate(make([]byte, 1024), 0)

	value := rval.Tocollate(clt).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("value2collate ... ok\n")
	return
}

func json2cbor2collate2value(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> value -> cbor -> collate -> value -> json -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	rval := config.NewValue(ref)
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	clt := config.NewCollate(make([]byte, 1024), 0)

	config.NewValue(rval.Tocbor(cbr).Tocollate(clt).Tovalue()).Tojson(jsn)
	_, value := jsn.Tovalue()
	// verify
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("json2cbor2collate2value ... ok\n")
	return
}

func json2collate2value(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> collate -> value -> json -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	jsn := config.NewJson(make([]byte, 1024), 0)
	clt := config.NewCollate(make([]byte, 1024), 0)

	val := config.NewValue(config.NewJson(data, -1).Tocollate(clt).Tovalue())
	_, value := val.Tojson(jsn).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("json2collate2value ... ok\n")
	return
}

func json2value(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> value -> json -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	rval := config.NewValue(ref)
	_, value := rval.Tojson(config.NewJson(make([]byte, 1024), 0)).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("json2value ... ok\n")
	return
}

func cbor2collate2value2json(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> cbor -> collate -> value -> json -> cbor
	_, ref := config.NewJson(data, -1).Tovalue()
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	clt := config.NewCollate(make([]byte, 1024), 0)
	cbrback := config.NewCbor(make([]byte, 1024), 0)

	value := config.NewJson(data, -1).Tocbor(cbr).Tocollate(clt).Tovalue()
	value = config.NewValue(value).Tojson(jsn).Tocbor(cbrback).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("cbor2collate2value2json ... ok\n")
	return
}

func cbor2value2json(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> cbor -> value -> json -> cbor -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	cbrback := config.NewCbor(make([]byte, 1024), 0)
	val := config.NewValue(config.NewJson(data, -1).Tocbor(cbr).Tovalue())
	value := val.Tojson(jsn).Tocbor(cbrback).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("cbor2value2json ... ok\n")
	return
}

func cbor2json(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> cbor -> json -> cbor -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	cbrback := config.NewCbor(make([]byte, 1024), 0)

	config.NewJson(data, -1).Tocbor(cbr).Tojson(jsn).Tocbor(cbrback)
	value := cbrback.Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("cbor2json ... ok\n")
	return
}

func collate2value2json2cbor(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> collate -> value -> json -> cbor -> collate -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	clt := config.NewCollate(make([]byte, 1024), 0)
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	cltback := config.NewCollate(make([]byte, 1024), 0)

	val := config.NewValue(config.NewJson(data, -1).Tocollate(clt).Tovalue())
	value := val.Tojson(jsn).Tocbor(cbr).Tocollate(cltback).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("collate2value2json2cbor ... ok\n")
	return
}

func collate2json2cbor(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> collate -> json -> cbor -> collate
	_, ref := config.NewJson(data, -1).Tovalue()
	clt := config.NewCollate(make([]byte, 1024), 0)
	jsn := config.NewJson(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	cltback := config.NewCollate(make([]byte, 1024), 0)

	config.NewJson(data, -1).Tocollate(clt).Tojson(jsn).Tocbor(cbr)
	value := cbr.Tocollate(cltback).Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("collate2json2cbor ... ok\n")
	return
}

func collate2cbor(config *gson.Config, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()
	// json -> collate -> cbor -> collate -> value
	_, ref := config.NewJson(data, -1).Tovalue()
	clt := config.NewCollate(make([]byte, 1024), 0)
	cbr := config.NewCbor(make([]byte, 1024), 0)
	cltback := config.NewCollate(make([]byte, 1024), 0)
	config.NewJson(data, -1).Tocollate(clt).Tocbor(cbr).Tocollate(cltback)
	value := cltback.Tovalue()
	ref, value = gson.Fixtojson(config, ref), gson.Fixtojson(config, value)
	if !reflect.DeepEqual(ref, value) {
		return fmt.Errorf("DeepEqual() expected %v, got %v", ref, value)
	}
	verbosef("collate2cbor ... ok\n")
	return
}

func verifyValuePointers(config *gson.Config, doc interface{}) (err error) {
	ndoc := cloneValue(config, doc)
	doc, ndoc = gson.Fixtojson(config, doc), gson.Fixtojson(config, ndoc)
	if !reflect.DeepEqual(doc, ndoc) {
		fmsg := "fail verifyValuePointers:\n  expected: %v, got %v\n"
		write(fmsg, doc, ndoc)
		return errors.New("fail verifyValuePointers")
	}
	verbosef("verifyValuePointers ... ok\n")
	return nil
}

func verifyCborPointers(config *gson.Config, docref interface{}) (err error) {
	// TODO: for now adjust the config options to unsupported parameters
	config = config.SetContainerEncoding(gson.Stream)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v\n", getStackTrace(2, debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()

	cbrout, err := cloneCbor(config, docref)
	if err != nil {
		return err
	}
	debugf(fmt.Sprintln("@cborClone", cbrout.Bytes()))
	doc := cbrout.Tovalue()
	doc, docref = gson.CborMap2golangMap(doc), gson.CborMap2golangMap(docref)
	doc, docref = gson.Fixtojson(config, doc), gson.Fixtojson(config, docref)
	if !reflect.DeepEqual(doc, docref) {
		write("cbor-gsp expected %v, got %v\n", docref, doc)
		return fmt.Errorf("error verifyCborPointers")
	}
	verbosef("verifyCborPointers ... ok\n")
	return nil
}

func generateJSON(prodfile string, seed, count int) chan string {
	bagdir := path.Dir(prodfile)
	text, err := ioutil.ReadFile(prodfile)
	if err != nil {
		log.Fatal(err)
	}
	root := compile(parsec.NewScanner(text)).(mcommon.Scope)
	scope := monster.BuildContext(root, uint64(seed), bagdir, prodfile)
	nterms := scope["_nonterminals"].(mcommon.NTForms)

	// compile monster production file.
	ch := make(chan string, 1000)
	mrand := rand.New(rand.NewSource(int64(seed)))
	go func() {
		nonterms := []string{
			"null", "bool", "integer", "float", "string", "s", "object",
		}
		for i := 0; i < count; i++ {
			nonterm := nonterms[mrand.Intn(len(nonterms))]
			scope = scope.RebuildContext()
			ch <- evaluate("root", scope, nterms[nonterm]).(string)
		}
		close(ch)
	}()
	return ch
}

func compile(s parsec.Scanner) parsec.ParsecNode {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v at %v", r, s.GetCursor())
		}
	}()
	root, _ := monster.Y(s)
	return root
}

func evaluate(
	name string, scope mcommon.Scope, forms []*mcommon.Form) interface{} {

	return monster.EvalForms(name, scope, forms)
}

func cloneValue(config *gson.Config, doc interface{}) interface{} {
	val := config.NewValue(doc)
	pointers := val.ListPointers([]string{})
	ptrs := make(jptrs, 0)
	for _, path := range pointers {
		ptrs = append(ptrs, config.NewJsonpointer(path))
	}
	sort.Sort(ptrs)

	// populate containers.
	var nval *gson.Value
	switch v := doc.(type) {
	case []interface{}:
		nval = config.NewValue(make([]interface{}, len(v)))
	case map[string]interface{}:
		nval = config.NewValue(make(map[string]interface{}))
	}
	for _, ptr := range ptrs {
		path := ptr.Path()

		if ln := len(path); ln == 0 || path[ln-1] == '-' {
			continue
		}
		switch v := val.Get(ptr).(type) {
		case []interface{}:
			nval.Set(ptr, make([]interface{}, len(v)))
		case map[string]interface{}:
			nval.Set(ptr, map[string]interface{}{})
		}
	}

	// populate values.
	for _, ptr := range ptrs {
		path := ptr.Path()
		if ln := len(path); ln == 0 || path[ln-1] == '-' {
			continue
		}
		switch v := val.Get(ptr).(type) {
		case []interface{}:
		case map[string]interface{}:
		default:
			nval.Set(ptr, v)
		}
	}
	return nval.Data()
}

func cloneCbor(config *gson.Config, doc interface{}) (*gson.Cbor, error) {
	val := config.NewValue(doc)
	cbr := val.Tocbor(config.NewCbor(make([]byte, 10*1024*1024), 0))

	pointers := val.ListPointers([]string{})
	ptrs := make(jptrs, 0)
	for _, path := range pointers {
		ptrs = append(ptrs, config.NewJsonpointer(path))
	}
	sort.Sort(ptrs)

	initArray := func(n int) []interface{} {
		arr := make([]interface{}, n)
		for i := 0; i < n; i++ {
			arr[i] = 0.0
		}
		return arr
	}

	// populate root container.
	out1 := config.NewCbor(make([]byte, 10*10*1024), 0)
	out2 := config.NewCbor(make([]byte, 10*10*1024), 0)
	switch v := doc.(type) {
	case []interface{}:
		config.NewValue(initArray(len(v))).Tocbor(out1)
	case [][2]interface{}:
		config.NewValue([][2]interface{}{}).Tocbor(out1)
	case map[string]interface{}:
		config.NewValue([][2]interface{}{}).Tocbor(out1)
	}

	debugf(fmt.Sprintln("@initial", out1.Bytes()))

	itemcbr := config.NewCbor(make([]byte, 10*1024*1024), 0)
	oitemcbr := config.NewCbor(make([]byte, 10*1024*1024), 0)

	// populate containers.
	for _, ptr := range ptrs {
		path := ptr.Path()
		debugf(fmt.Sprintln("@jptrc", string(path)))
		if ln := len(path); ln == 0 || path[ln-1] == '-' {
			continue
		}
		value := cbr.Get(ptr, itemcbr.Reset(nil)).Tovalue()
		switch v := value.(type) {
		case []interface{}:
			debugf(fmt.Sprintln("@insertc"))
			config.NewValue(initArray(len(v))).Tocbor(itemcbr.Reset(nil))
			out1.Set(ptr, itemcbr, out2.Reset(nil), oitemcbr.Reset(nil))
			debugf(fmt.Sprintln("@contain", out2.Bytes()))
			out1, out2 = out2, out1

		case map[string]interface{}:
			debugf(fmt.Sprintln("@insertm"))
			config.NewValue(map[string]interface{}{}).Tocbor(itemcbr.Reset(nil))
			out1.Set(ptr, itemcbr, out2.Reset(nil), oitemcbr.Reset(nil))
			debugf(fmt.Sprintln("@contain", out2.Bytes()))
			out1, out2 = out2, out1
		}
	}

	debugf(fmt.Sprintln("@cntnrdc", out1.Bytes()))

	// populate values.
	for _, ptr := range ptrs {
		path := ptr.Path()
		if ln := len(path); ln == 0 || path[ln-1] == '-' {
			continue
		}
		debugf(fmt.Sprintln("@jptrv", string(path)))
		value := cbr.Get(ptr, itemcbr.Reset(nil)).Tovalue()
		switch value.(type) {
		case []interface{}:
		case map[string]interface{}:
		default:
			debugf(fmt.Sprintln("@insertv"))
			out1.Set(ptr, itemcbr, out2.Reset(nil), oitemcbr.Reset(nil))
			debugf(fmt.Sprintln("@contain", out2.Bytes()))
			out1, out2 = out2, out1
		}
	}
	return out1, nil
}

func makeConfig(mrand *rand.Rand) *gson.Config {
	config := gson.NewDefaultConfig()
	nks := []string{"smart", "float"}
	nk := nks[mrand.Intn(len(nks))]
	switch nk {
	case "smart":
		config = config.SetNumberKind(gson.SmartNumber)
		incrparam("SmartNumber", 1)
	case "float":
		config = config.SetNumberKind(gson.FloatNumber)
		incrparam("FloatNumber", 1)
	}

	wss := []string{"ansi", "unicode"}
	ws := wss[mrand.Intn(len(wss))]
	switch ws {
	case "ansi":
		config = config.SetSpaceKind(gson.AnsiSpace)
		incrparam("AnsiSpace", 1)
	case "unicode":
		config = config.SetSpaceKind(gson.UnicodeSpace)
		incrparam("UnicodeSpace", 1)
	}

	cts := []string{"lenprefix", "stream"}
	ct := cts[mrand.Intn(len(cts))]
	switch ct {
	case "lenprefix":
		config = config.SetContainerEncoding(gson.LengthPrefix)
		incrparam("LengthPrefix", 1)
	case "stream":
		config = config.SetContainerEncoding(gson.Stream)
		incrparam("Stream", 1)
	}

	bools := []bool{true, false}

	sortbyarraylen, sortbyproplen := bools[mrand.Intn(2)], bools[mrand.Intn(2)]
	config = config.SortbyArrayLen(sortbyarraylen)
	config = config.SortbyPropertyLen(sortbyproplen)
	if sortbyarraylen {
		incrparam("arrayLenPrefix", 1)
	}
	if sortbyproplen {
		incrparam("propertyLenPrefix", 1)
	}

	missing := bools[mrand.Intn(2)]
	config = config.UseMissing(missing).SetStrict(false)
	if missing {
		incrparam("doMissing", 1)
	}
	//if strict {
	//	incrparam("strict", 1)
	//}
	return config
}

func incrparam(param string, delta int) {
	statrw.Lock()
	defer statrw.Unlock()
	statistics[param] = statistics[param].(int) + delta
}

func bookstats(config *gson.Config, js string, doc interface{}, err error) {
	if err != nil {
		incrparam("fail", 1)
	} else {
		incrparam("pass", 1)
	}
	incrparam("bytes", len(js))
	switch v := gson.Fixtojson(config, doc).(type) {
	case nil:
		incrparam("null", 1)
	case bool:
		if v {
			incrparam("true", 1)
		} else {
			incrparam("false", 1)
		}
	case int64, float64, float32:
		incrparam("num", 1)
	case string:
		incrparam("string", 1)
	case []interface{}:
		incrparam("array", 1)
	case map[string]interface{}:
		incrparam("object", 1)
	default:
		panic(fmt.Errorf("uknown type %T\n", v))
	}
}

func printStatistics() {
	statrw.RLock()
	defer statrw.RUnlock()

	write("seed: %v\n", options.seed)
	keys := []string{"docs", "pass", "fail", "bytes"}
	properties := []string{}
	for _, key := range keys {
		value := statistics[key]
		properties = append(properties, fmt.Sprintf("%v: %v", key, value))
	}
	write(strings.Join(properties, ", ") + "\n")

	keys = []string{"null", "true", "false", "num", "string", "array", "object"}
	properties = []string{}
	for _, key := range keys {
		value := statistics[key]
		properties = append(properties, fmt.Sprintf("%v: %v", key, value))
	}
	write(strings.Join(properties, ", ") + "\n")

	keys = []string{"FloatNumber", "SmartNumber"}
	properties = []string{}
	for _, key := range keys {
		value := statistics[key]
		properties = append(properties, fmt.Sprintf("%v: %v", key, value))
	}
	write(strings.Join(properties, ", ") + "\n")

	keys = []string{"AnsiSpace", "UnicodeSpace"}
	properties = []string{}
	for _, key := range keys {
		value := statistics[key]
		properties = append(properties, fmt.Sprintf("%v: %v", key, value))
	}
	write(strings.Join(properties, ", ") + "\n")

	keys = []string{"LengthPrefix", "Stream"}
	properties = []string{}
	for _, key := range keys {
		value := statistics[key]
		properties = append(properties, fmt.Sprintf("%v: %v", key, value))
	}
	write(strings.Join(properties, ", ") + "\n")

	properties = []string{}
	keys = []string{"arrayLenPrefix", "propertyLenPrefix", "doMissing", "strict"}
	for _, key := range keys {
		value := statistics[key]
		properties = append(properties, fmt.Sprintf("%v: %v", key, value))
	}
	write(strings.Join(properties, ", ") + "\n")
}

func printFailure(config *gson.Config, fmsg string, err error, inp string) {
	write("seed   : %v\n", options.seed)
	write("config : %v\n", config.String())
	write(fmsg, err, inp)
}

func bytes2str(bytes []byte) string {
	if bytes == nil {
		return ""
	}
	sl := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	st := &reflect.StringHeader{Data: sl.Data, Len: sl.Len}
	return *(*string)(unsafe.Pointer(st))
}

func str2bytes(str string) []byte {
	if str == "" {
		return nil
	}
	st := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sl := &reflect.SliceHeader{Data: st.Data, Len: st.Len, Cap: st.Len}
	return *(*[]byte)(unsafe.Pointer(sl))
}

func getStackTrace(skip int, stack []byte) string {
	var buf bytes.Buffer
	lines := strings.Split(string(stack), "\n")
	for _, call := range lines[skip*2:] {
		buf.WriteString(fmt.Sprintf("%s\n", call))
	}
	return buf.String()
}

func isArrayOffset(ptr string, container interface{}) (key string, array bool) {
	xs := strings.Split(ptr, "/")
	key = xs[len(xs)-1]
	if _, ok := container.([]interface{}); ok {
		return key, true
	}
	return key, false
}

func write(fmsg string, args ...interface{}) {
	s := fmt.Sprintf(fmsg, args...)
	if options.genout != "" {
		options.outfd.Write([]byte(s))
	} else {
		fmt.Printf(s)
	}
}

func verbosef(fmsg string, args ...interface{}) {
	if options.verbose || options.debug {
		write(fmsg, args...)
	}
}

func debugf(fmsg string, args ...interface{}) {
	if options.debug {
		write(fmsg, args...)
	}
}
