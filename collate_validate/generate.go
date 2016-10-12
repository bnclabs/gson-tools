package main

import "io/ioutil"
import "log"
import "fmt"
import "math/rand"
import "path"
import "strconv"
import "encoding/json"

import "github.com/prataprc/goparsec"
import "github.com/prataprc/monster"
import mcommon "github.com/prataprc/monster/common"

func randInteger(mrand *rand.Rand) int {
	x := mrand.Int() % 1000000000
	if (x % 3) == 0 {
		return -x
	}
	return x
}

func generateInteger(seed, count int, ch chan string) {
	mrand := rand.New(rand.NewSource(int64(seed)))
	for i := 0; i < count; i++ {
		ch <- strconv.Itoa(randInteger(mrand))
	}
}

func generateSD(seed, count int, ch chan string) {
	mrand := rand.New(rand.NewSource(int64(seed)))
	for i := 0; i < count; i++ {
		x := randInteger(mrand)
		ch <- strconv.FormatFloat(1/float64(x+1), 'f', -1, 64)
	}
}

func generateLD(seed, count int, ch chan string) {
	mrand := rand.New(rand.NewSource(int64(seed)))
	for i := 0; i < count; i++ {
		x := randInteger(mrand)
		y := float64(x)/float64(mrand.Int()+1) + 1
		if -1.0 < y && y < 1.0 {
			y += 2
		}
		ch <- strconv.FormatFloat(y, 'f', -1, 64)
	}
}

func generateFloats(seed, count int, ch chan string) {
	mrand := rand.New(rand.NewSource(int64(seed)))
	for i := 0; i < count; i++ {
		x := randInteger(mrand)
		f := float64(x) / float64(mrand.Int()+1)
		ch <- strconv.FormatFloat(f, 'e', -1, 64)
	}
}

func generateJSON(prodfile string, seed, count int, ch chan string) {
	mrand := rand.New(rand.NewSource(int64(seed)))
	bagdir := path.Dir(prodfile)
	text, err := ioutil.ReadFile(prodfile)
	if err != nil {
		log.Fatal(err)
	}
	root := compile(parsec.NewScanner(text)).(mcommon.Scope)
	scope := monster.BuildContext(root, uint64(seed), bagdir, prodfile)
	nterms := scope["_nonterminals"].(mcommon.NTForms)

	nonterms := []string{
		"null", "bool", "integer", "float", "string", "s", "object",
	}
	// compile monster production file.
	var val interface{}
	for i := 0; i < count; i++ {
		nonterm := nonterms[mrand.Intn(len(nonterms))]
		scope = scope.RebuildContext()
		jsons := evaluate("root", scope, nterms[nonterm]).(string)
		if err := json.Unmarshal([]byte(jsons), &val); err != nil {
			fmt.Printf("json: %v\n", jsons)
			panic(err)
		}
		outs, err := json.Marshal(val)
		if err != nil {
			fmt.Printf("json: %v\n", jsons)
			panic(err)
		}
		ch <- string(outs)
	}
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

func evaluate(name string, scope mcommon.Scope, forms []*mcommon.Form) interface{} {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v", r)
		}
	}()
	return monster.EvalForms(name, scope, forms)
}
