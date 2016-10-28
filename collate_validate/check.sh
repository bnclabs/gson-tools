#! /usr/bin/env bash

go build
GOMAXPROCS=16 ./collate_validate -repeat 100 -count 10000 -seed 1591398756310399222
