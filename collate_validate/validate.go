package main

import "flag"
import "os"
import "bytes"
import "strings"
import "sync"
import "fmt"
import "math/rand"
import "runtime/debug"
import "sort"
import "time"
import "path"
import "runtime"
import "reflect"

import qv "github.com/couchbase/query/value"
import "github.com/prataprc/gson"

var options struct {
	repeat   int
	count    int
	seed     int
	prodfile string
}

func argParse() []string {
	flag.IntVar(&options.repeat, "repeat", 1,
		"number of times to repeat the sort")
	flag.IntVar(&options.count, "count", 1,
		"number of items to sort")
	flag.IntVar(&options.seed, "seed", 0,
		"random seed to monster")
	flag.StringVar(&options.prodfile, "prodfile", "",
		"random seed to monster")
	flag.Parse()

	if options.seed == 0 {
		options.seed = rand.Int()
	}

	if options.prodfile == "" {
		_, filename, _, _ := runtime.Caller(0)
		options.prodfile = path.Join(path.Dir(filename), "json.prod")
	}

	return flag.Args()
}

func main() {
	argParse()
	for i := 0; i < options.repeat; i++ {
		collateValidate(options.seed + i)
		fmt.Println()
	}
}

func collateValidate(seed int) {
	count1to4 := options.count / 5
	count5 := options.count - (count1to4 * 4)
	fmt.Printf("seed      : %v\n", seed)
	fmt.Printf("items     : %v %v %v\n", options.count, count1to4, count5)

	generate := func(ch chan string) {
		generateInteger(seed, count1to4, ch)
		generateSD(seed, count1to4, ch)
		generateLD(seed, count1to4, ch)
		generateFloats(seed, count1to4, ch)
		generateJSON(options.prodfile, seed, count5, ch)
	}

	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		mrand := rand.New(rand.NewSource(int64(seed)))
		config := makeConfig(mrand)
		ch := make(chan string, 1000)
		go func() { generate(ch); close(ch) }()
		validateWith(
			config,
			"JsonToValueToCborToCollate",
			options.count,
			ch,
			func(input []byte) []byte {
				cbr := config.NewCbor(make([]byte, 1024), 0)
				clt := config.NewCollate(make([]byte, 1024), 0)
				_, value := config.NewJson(input, -1).Tovalue()
				return config.NewValue(value).Tocbor(cbr).Tocollate(clt).Bytes()
			})
		wg.Done()
	}()

	go func() { // JsonToValueToCollate
		mrand := rand.New(rand.NewSource(int64(seed)))
		config := makeConfig(mrand)
		ch := make(chan string, 1000)
		go func() { generate(ch); close(ch) }()
		validateWith(
			config,
			"JsonToValueToCollate",
			options.count,
			ch,
			func(input []byte) []byte {
				clt := config.NewCollate(make([]byte, 1024), 0)
				_, value := config.NewJson(input, -1).Tovalue()
				return config.NewValue(value).Tocollate(clt).Bytes()
			})
		wg.Done()
	}()

	go func() { // JsonToCollate
		mrand := rand.New(rand.NewSource(int64(seed)))
		config := makeConfig(mrand)
		ch := make(chan string, 1000)
		go func() { generate(ch); close(ch) }()
		validateWith(
			config,
			"JsonToCollate",
			options.count,
			ch,
			func(input []byte) []byte {
				clt := config.NewCollate(make([]byte, 1024), 0)
				return config.NewJson(input, -1).Tocollate(clt).Bytes()
			})
		wg.Done()
	}()

	go func() {
		mrand := rand.New(rand.NewSource(int64(seed)))
		config := makeConfig(mrand)
		ch := make(chan string, 1000)
		go func() { generate(ch); close(ch) }()
		validateWith(
			config,
			"JsonToCborToValueToCollate",
			options.count,
			ch,
			func(input []byte) []byte {
				cbr := config.NewCbor(make([]byte, 1024), 0)
				clt := config.NewCollate(make([]byte, 1024), 0)
				value := config.NewJson(input, -1).Tocbor(cbr).Tovalue()
				return config.NewValue(value).Tocollate(clt).Bytes()
			})
		wg.Done()
	}()

	go func() {
		mrand := rand.New(rand.NewSource(int64(seed)))
		config := makeConfig(mrand)
		ch := make(chan string, 1000)
		go func() { generate(ch); close(ch) }()
		validateWith(
			config,
			"JsonToCborToCollate",
			options.count,
			ch,
			func(input []byte) []byte {
				cbr := config.NewCbor(make([]byte, 1024), 0)
				clt := config.NewCollate(make([]byte, 1024), 0)
				config.NewJson(input, -1).Tocbor(cbr)
				return cbr.Tocollate(clt).Bytes()
			})
		wg.Done()
	}()

	wg.Wait()
}

func validateWith(
	config *gson.Config, nm string, count int, ch chan string,
	fn func([]byte) []byte) {

	var input string

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %v\n", r)
			fmt.Printf("%v", getStackTrace(2, debug.Stack()))
			fmt.Printf("json : %q\n", input)
		}
	}()

	inputs := make([]string, 0, count)
	collated := make([][]byte, 0, count)
	for input = range ch {
		inputs = append(inputs, input)
		collated = append(collated, fn([]byte(input)))
	}

	rawlist := &jsonList{vals: inputs, compares: 0}
	rawts := timeIt(func() { sort.Sort(rawlist) })
	bints := timeIt(func() { sort.Sort(ByteSlices(collated)) })
	fmt.Printf("config: %v\n", config.String())
	fmsg := "%-30v: %v Vs %v %v compares\n"
	fmt.Printf(fmsg, nm, rawts, bints, rawlist.compares)

	// validate sort order
	refs := make([]interface{}, 0, count)
	jsn := config.NewJson(make([]byte, 1024), 0)
	for _, input = range rawlist.vals {
		_, ref := jsn.Reset([]byte(input)).Tovalue()
		refs = append(refs, gson.Fixtojson(config, ref))
	}

	values := make([]interface{}, 0, count)
	clt := config.NewCollate(make([]byte, 1024), 0)
	jsn = config.NewJson(make([]byte, 1024), 0)
	for _, collin := range collated {
		_, value := clt.Reset([]byte(collin)).Tojson(jsn.Reset(nil)).Tovalue()
		values = append(values, gson.Fixtojson(config, value))
	}

	// check
	exit := false
	if n, m := len(values), len(refs); n != m {
		fmt.Printf("expected %v, got %v\n", m, n)
		exit = true
	}
	for i, ref := range refs {
		if !reflect.DeepEqual(ref, values[i]) {
			fmt.Printf("index %v expected %v, got %v\n", i, ref, values[i])
			exit = true
		}
	}
	if exit {
		os.Exit(1)
	}
}

func makeConfig(mrand *rand.Rand) *gson.Config {
	config := gson.NewDefaultConfig()
	nks := []string{"f64"}
	nk := nks[mrand.Intn(len(nks))]
	switch nk {
	case "snum":
		// TODO: we are not going to test for smartnumbers.
		// config = config.SetNumberKind(gson.SmartNumber)
		config = config.SetNumberKind(gson.FloatNumber)
		//incrparam("FloatNumber", 1)
	case "snum32":
		// TODO: we are not going to test for smartnumbers32.
		// config = config.SetNumberKind(gson.SmartNumber32)
		config = config.SetNumberKind(gson.FloatNumber)
		//incrparam("FloatNumber", 1)
	case "int":
		config = config.SetNumberKind(gson.IntNumber).SetStrict(false)
		//incrparam("IntNumber", 1)
	case "f64":
		config = config.SetNumberKind(gson.FloatNumber)
		//incrparam("FloatNumber", 1)
	case "f32":
		config = config.SetNumberKind(gson.FloatNumber32)
		//incrparam("FloatNumber32", 1)
	case "dec":
		// TODO: we are not going to test for decimal numbers.
		// config = config.SetNumberKind(gson.Decimal)
		config = config.SetNumberKind(gson.FloatNumber)
		//incrparam("FloatNumber", 1)
	}
	wss := []string{"ansi", "unicode"}
	ws := wss[mrand.Intn(len(wss))]
	switch ws {
	case "ansi":
		config = config.SetSpaceKind(gson.AnsiSpace)
		//incrparam("AnsiSpace", 1)
	case "unicode":
		config = config.SetSpaceKind(gson.UnicodeSpace)
		//incrparam("UnicodeSpace", 1)
	}
	cts := []string{"lenprefix", "stream"}
	ct := cts[mrand.Intn(len(cts))]
	switch ct {
	case "lenprefix":
		config = config.SetContainerEncoding(gson.LengthPrefix)
		//incrparam("LengthPrefix", 1)
	case "stream":
		config = config.SetContainerEncoding(gson.Stream)
		//incrparam("Stream", 1)
	}

	bools := []bool{true, false}

	sortbyarraylen := bools[mrand.Intn(2)]
	config = config.SortbyArrayLen(sortbyarraylen)
	//if sortbyarraylen {
	//	incrparam("arrayLenPrefix", 1)
	//}
	//if sortbyproplen {
	//	incrparam("propertyLenPrefix", 1)
	//}

	missing := bools[mrand.Intn(2)]
	config = config.UseMissing(missing).SetStrict(false)
	if missing {
		//incrparam("doMissing", 1)
	}
	//if strict {
	//	incrparam("strict", 1)
	//}
	return config
}

func timeIt(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

func getStackTrace(skip int, stack []byte) string {
	var buf bytes.Buffer
	lines := strings.Split(string(stack), "\n")
	for _, call := range lines[skip*2:] {
		buf.WriteString(fmt.Sprintf("%s\n", call))
	}
	return buf.String()
}

// sort type for n1ql

type jsonList struct {
	compares int
	vals     []string
}

func (jsons *jsonList) Len() int {
	return len(jsons.vals)
}

func (jsons *jsonList) Less(i, j int) bool {
	key1, key2 := jsons.vals[i], jsons.vals[j]
	jsons.compares++
	value1 := qv.NewValue([]byte(key1))
	value2 := qv.NewValue([]byte(key2))
	return value1.Collate(value2) < 0
}

func (jsons *jsonList) Swap(i, j int) {
	jsons.vals[i], jsons.vals[j] = jsons.vals[j], jsons.vals[i]
}

// sort type for slice of []byte

type ByteSlices [][]byte

func (bs ByteSlices) Len() int {
	return len(bs)
}

func (bs ByteSlices) Less(i, j int) bool {
	return bytes.Compare(bs[i], bs[j]) < 0
}

func (bs ByteSlices) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}
