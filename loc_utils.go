package main

import (
	"bufio"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

type coord struct {
	Lat  float64
	Long float64
}

var zipCodeToLatLong map[string]coord

func init() {
	zipCodeToLatLong = make(map[string]coord, 0)
	f, err := os.Open("./assets/zipCodeData.txt")
	if err != nil {
		log.Fatalf("failed to open zip code data file: %v", err)
	}
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		data := strings.Split(scanner.Text(), "\t")
		zipcode := data[1]
		lat, _ := strconv.ParseFloat(data[9], 64)
		long, _ := strconv.ParseFloat(data[10], 64)
		zipCodeToLatLong[zipcode] = coord{Lat: lat, Long: long}
	}
	log.Println("successfully parsed zip code data")
}

// Distance calculates distance in miles between two points.
// Copied from https://www.geodatasource.com/developers/go under LGPLv3 licensing.
// See https://choosealicense.com/licenses/gpl-3.0.
func Distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515

	return dist
}
