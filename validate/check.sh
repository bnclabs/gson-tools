#! /usr/bin/env bash
go build

runinput() {
    echo $1 $2
    $1 "$2"
    rc=$?
    if [[ $rc != 0 ]]; then
        echo "failed $1 $2"
        exit $rc;
    fi
    echo ""
}

runinput './validate -count 10000 -input' 'null'
runinput './validate -count 10000 -input' 'true'
runinput './validate -count 10000 -input' 'false'

runinput './validate -count 10000 -input' '10'
runinput './validate -count 10000 -input' '-10'
runinput './validate -count 10000 -input' '1000'
runinput './validate -count 10000 -input' '-1000'
runinput './validate -count 10000 -input' '100000'
runinput './validate -count 10000 -input' '-100000'
runinput './validate -count 10000 -input' '2000000000'
runinput './validate -count 10000 -input' '-2000000000'
runinput './validate -count 10000 -input' '-20000000000'
runinput './validate -count 10000 -input' '20000000000'

runinput './validate -count 10000 -input' '10.12334562342343'
runinput './validate -count 10000 -input' '-10.12332342345643'
runinput './validate -count 10000 -input' '1000.12334564234233'
runinput './validate -count 10000 -input' '-1000.12334564323423'
runinput './validate -count 10000 -input' '100000.12334523423643'
runinput './validate -count 10000 -input' '-100000.12334564323423'
runinput './validate -count 10000 -input' '2000000000.12334564323423'
runinput './validate -count 10000 -input' '-2000000000.12334564323423'
runinput './validate -count 10000 -input' '-20000000000.12334564323423'
runinput './validate -count 10000 -input' '20000000000.12334564323423'

runinput './validate -count 10000 -input' '1'
runinput './validate -count 10000 -input' '0.123456789123'
runinput './validate -count 10000 -input' '-0.123456789123'
runinput './validate -count 10000 -input' '10.1'
runinput './validate -count 10000 -input' '-10.1'
runinput './validate -count 10000 -input' '-10E-1'
runinput './validate -count 10000 -input' '-10e+1'
runinput './validate -count 10000 -input' '10E-1'
runinput './validate -count 10000 -input' '10e+1'

runinput './validate -count 10000 -input' '"true"'
runinput './validate -count 10000 -input' '"tru\"e"'
runinput './validate -count 10000 -input' '"tru\e"'
runinput './validate -count 10000 -input' '"tru\be"'
runinput './validate -count 10000 -input' '"tru\fe"'
runinput './validate -count 10000 -input' '"tru\ne"'
runinput './validate -count 10000 -input' '"tru\re"'
runinput './validate -count 10000 -input' '"tru\te"'
runinput './validate -count 10000 -input' '"null"'
runinput './validate -count 10000 -input' '"\n true "'
runinput './validate -count 10000 -input' '"\t 1 "'
runinput './validate -count 10000 -input' '"\r 1.2 "'
runinput './validate -count 10000 -input' '"\t -5 \n"'
runinput './validate -count 10000 -input' '"\t \"a\u1234\" \n"'
runinput './validate -count 10000 -input' '"tru\u0123e"'
runinput './validate -count 10000 -input' '"汉语 / 漢語; Hàn\b \t\uef24yǔ "'
runinput './validate -count 10000 -input' '"a\u1234"'
runinput './validate -count 10000 -input' '"http:\/\/"'
runinput './validate -count 10000 -input' '"invalid: \uD834x\uDD1E"'
runinput './validate -count 10000 -input' '"\"foobar\"\u003chtml\u003e [\u2028 \u2029]"'
runinput './validate -count 10000 -input' '"hello\\\ud800world"'
runinput './validate -count 10000 -input' '"hello\ud800\\\ud800world"'
runinput './validate -count 10000 -input' '"hello\ud800\ud800world"'

runinput './validate -count 3000 -input' '[  ]'
runinput './validate -count 3000 -input' '[]'
runinput './validate -count 3000 -input' '[ null, true, false, 10, "tru\"e"]'
runinput './validate -count 3000 -input' '[{}]'
runinput './validate -count 3000 -input' '[{"T":false}]'
runinput './validate -count 3000 -input' '[{"T":false}]'
runinput './validate -count 3000 -input' '[1, 2, 3]'

runinput './validate -count 3000 -input' '{  }'
runinput './validate -count 3000 -input' '{"X": [1,2,3], "Y": 4}'
runinput './validate -count 3000 -input' '{"x": 1}'
runinput './validate -count 3000 -input' '{"F1":1,"F2":2,"F3":3}'
runinput './validate -count 3000 -input' '{"k1":1,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}'
runinput './validate -count 3000 -input' '{"k1":1,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}'
runinput './validate -count 3000 -input' '{"Y": 1, "Z": 2}'
runinput './validate -count 3000 -input' '{"alpha": "abc", "alphabet": "xyz"}'
runinput './validate -count 3000 -input' '{"alpha": "abc"}'
runinput './validate -count 3000 -input' '{"alphabet": "xyz"}'
runinput './validate -count 3000 -input' '{"T":[]}'
runinput './validate -count 3000 -input' '{"T":null}'
runinput './validate -count 3000 -input' '{"T":false}'
runinput './validate -count 3000 -input' '{"T":false}'
runinput './validate -count 3000 -input' '{"M":{"T":false}}'
runinput './validate -count 3000 -input' '{"2009-11-10T23:00:00Z": "hello world"}'
runinput './validate -count 3000 -input' '{ "a": null, "b" : true,"c":false, "d\"":10, "e":"tru\"e" }'
runinput './validate -count 3000 -input' '{"resurvey":true,"2":{},"breasted":"overrecord"}'
runinput './validate -count 3000 -input' '{"inopportuneness":{},"/i\\j":[-56.741217148673634,"agalwood",-74555,"Heliotropium",-2.6370960188883714],"saddlebow":false}'
runinput './validate -count 3000 -input' '{"boatbuilding":false,"g~1n~1r":[5.67634687693652,{},"polyphalangism",8508,57906],"weibyeite":27.482278930827377}'
runinput './validate -count 3000 -input' '{"g~1n~1r":[58.433721717200484,false,"arsenophagy",{},-43570]}'
runinput './validate -count 3000 -input' '{"a~1b":["neoimpressionist",{},34.581719871452094,-78367,true]}'
GOMAXPROCS=16 ./validate -par 8 -count 20000
