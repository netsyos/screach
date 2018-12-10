package lib

import (
	"context"
	"log"

	"github.com/kr/pretty"
	"googlemaps.github.io/maps"
)

type MapAPI struct {
	Key          string
	Country      string
	Destinations []string
	Client       *maps.Client
}

func (m *MapAPI) Init() {
	c, err := maps.NewClient(maps.WithAPIKey(m.Key))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}
	m.Client = c
}

func (m *MapAPI) GeoCode(Address string) []maps.GeocodingResult {

	r := &maps.GeocodingRequest{
		Address: Address + ", " + m.Country,
	}
	result, err := m.Client.Geocode(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	pretty.Println(result)
	return result
}

func (m *MapAPI) GetDirection(Origin string, Destination string) []maps.Route {

	r := &maps.DirectionsRequest{
		Origin:      Origin + ", " + m.Country,
		Destination: Destination + ", " + m.Country,
	}
	route, _, err := m.Client.Directions(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	pretty.Println(route)
	return route
}

func (m *MapAPI) GetTravelTime(Origin string, Destination string) *maps.DistanceMatrixResponse {
	r := &maps.DistanceMatrixRequest{
		Origins:      []string{Origin + ", " + m.Country},
		Destinations: []string{Destination + ", " + m.Country},
	}
	result, err := m.Client.DistanceMatrix(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	pretty.Println(result)
	return result
}
