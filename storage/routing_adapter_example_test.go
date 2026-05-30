package storage_test

import (
	"fmt"
	"time"

	"github.com/MickMake/GoTravel/storage"
)

func ExampleMatchTraceRequestFromPoints() {
	points := []storage.Point{
		{DT: time.Unix(1700000000, 0), Lat: -33.8, Lng: 151.2},
		{DT: time.Unix(1700000060, 0), Lat: -33.9, Lng: 151.3},
	}

	request, err := storage.MatchTraceRequestFromPoints(points, storage.MatchTraceOptions{Profile: "driving"})
	if err != nil {
		panic(err)
	}

	fmt.Println(request.Profile)
	fmt.Println(len(request.Points))

	// Output:
	// driving
	// 2
}
