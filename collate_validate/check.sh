#! /usr/bin/env bash

go build
GOMAXPROCS=16 ./collate_validate -seed 3627207332921908491 -repeat 100 -count 10000
