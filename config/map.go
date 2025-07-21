package config

import (
	"encoding/csv"
	"io"
	"log"
	"strconv"
	"strings"
)

var Maps = []Map{}

type Map [][]int

func pushMap(m []byte) {
	r := csv.NewReader(strings.NewReader(string(m)))
	r.Comma = ';'
	var matrix Map
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		row := make([]int, len(record))
		for i, val := range record {
			n, err := strconv.Atoi(val)
			if err != nil {
				panic(err)
			}
			row[i] = n
		}
		matrix = append(matrix, row)
	}
	Maps = append(Maps, matrix)
}

func initMap() {
	pushMap(map1)
}
