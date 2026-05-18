package rules

import "fmt"

// City represents a location on the Eldritch Horror global map.
type City string

// Cities on the global map organized by continent.
const (
	// Americas
	CityArkham       City = "arkham"
	CitySanFrancisco City = "sanFrancisco"
	CityBuenosAires  City = "buenosAires"

	// Europe
	CityLondon   City = "london"
	CityRome     City = "rome"
	CityIstanbul City = "istanbul"

	// Asia
	CityTokyo    City = "tokyo"
	CityShanghai City = "shanghai"
	CityBangkok  City = "bangkok"

	// Africa
	CityCairo    City = "cairo"
	CityTunis    City = "tunis"
	CityCapeTown City = "capeTown"

	// Pacific
	CitySydney   City = "sydney"
	CityHonolulu City = "honolulu"

	// Middle East / Asia Minor
	CityJerusalem City = "jerusalem"

	// Additional major cities
	CityPyramids         City = "pyramids"      // Special location near Cairo
	CityAntarctica       City = "antarctica"    // Remote expedition location
	CityAmazon           City = "amazon"        // Remote expedition location
	CityHimalayas        City = "himalayas"     // Remote expedition location
	CityTheHeartOfAfrica City = "heartOfAfrica" // Remote expedition location
)

// TravelRoute represents a connection between two cities.
type TravelRoute struct {
	From City
	To   City
	Cost int // Action cost to travel this route
	Type RouteType
}

// RouteType indicates the travel method required.
type RouteType string

const (
	RouteTypeLand  RouteType = "land"  // No ticket required
	RouteTypeShip  RouteType = "ship"  // Requires ship ticket
	RouteTypeTrain RouteType = "train" // Requires train ticket
)

// GlobalMap defines the Eldritch Horror world map with cities and travel routes.
type GlobalMap struct {
	cities map[City]bool
	routes []TravelRoute
}

// NewGlobalMap creates the standard Eldritch Horror global map.
func NewGlobalMap() *GlobalMap {
	gm := &GlobalMap{
		cities: make(map[City]bool),
		routes: []TravelRoute{},
	}

	// Register all cities
	cities := []City{
		CityArkham, CitySanFrancisco, CityBuenosAires,
		CityLondon, CityRome, CityIstanbul,
		CityTokyo, CityShanghai, CityBangkok,
		CityCairo, CityTunis, CityCapeTown,
		CitySydney, CityHonolulu,
		CityJerusalem, CityPyramids,
		CityAntarctica, CityAmazon, CityHimalayas, CityTheHeartOfAfrica,
	}
	for _, city := range cities {
		gm.cities[city] = true
	}

	// Define travel routes (bidirectional)
	// Americas network
	gm.addBidirectionalRoute(CityArkham, CitySanFrancisco, 2, RouteTypeTrain)
	gm.addBidirectionalRoute(CitySanFrancisco, CityHonolulu, 2, RouteTypeShip)
	gm.addBidirectionalRoute(CityArkham, CityLondon, 2, RouteTypeShip)
	gm.addBidirectionalRoute(CitySanFrancisco, CityBuenosAires, 3, RouteTypeShip)
	gm.addBidirectionalRoute(CityBuenosAires, CityAmazon, 2, RouteTypeLand)

	// Europe network
	gm.addBidirectionalRoute(CityLondon, CityRome, 2, RouteTypeTrain)
	gm.addBidirectionalRoute(CityRome, CityIstanbul, 2, RouteTypeTrain)
	gm.addBidirectionalRoute(CityLondon, CityIstanbul, 3, RouteTypeShip)

	// Asia network
	gm.addBidirectionalRoute(CityIstanbul, CityShanghai, 3, RouteTypeShip)
	gm.addBidirectionalRoute(CityShanghai, CityTokyo, 2, RouteTypeShip)
	gm.addBidirectionalRoute(CityShanghai, CityBangkok, 2, RouteTypeTrain)
	gm.addBidirectionalRoute(CityTokyo, CityHonolulu, 2, RouteTypeShip)
	gm.addBidirectionalRoute(CityBangkok, CityHimalayas, 2, RouteTypeLand)

	// Africa network
	gm.addBidirectionalRoute(CityRome, CityCairo, 2, RouteTypeShip)
	gm.addBidirectionalRoute(CityCairo, CityPyramids, 1, RouteTypeLand)
	gm.addBidirectionalRoute(CityCairo, CityJerusalem, 1, RouteTypeTrain)
	gm.addBidirectionalRoute(CityCairo, CityTunis, 2, RouteTypeShip)
	gm.addBidirectionalRoute(CityCairo, CityCapeTown, 3, RouteTypeShip)
	gm.addBidirectionalRoute(CityCapeTown, CityTheHeartOfAfrica, 2, RouteTypeLand)

	// Pacific network
	gm.addBidirectionalRoute(CityHonolulu, CitySydney, 3, RouteTypeShip)
	gm.addBidirectionalRoute(CitySydney, CityAntarctica, 3, RouteTypeShip)

	// Cross-ocean routes
	gm.addBidirectionalRoute(CityBuenosAires, CityCapeTown, 3, RouteTypeShip)
	gm.addBidirectionalRoute(CityLondon, CityArkham, 2, RouteTypeShip)

	return gm
}

// addBidirectionalRoute adds a route in both directions.
func (gm *GlobalMap) addBidirectionalRoute(city1, city2 City, cost int, routeType RouteType) {
	gm.routes = append(gm.routes,
		TravelRoute{From: city1, To: city2, Cost: cost, Type: routeType},
		TravelRoute{From: city2, To: city1, Cost: cost, Type: routeType},
	)
}

// IsValidCity checks if a city exists on the map.
func (gm *GlobalMap) IsValidCity(city City) bool {
	return gm.cities[city]
}

// CanTravelTo checks if direct travel from one city to another is possible.
// Returns true if a route exists, regardless of ticket requirements.
func (gm *GlobalMap) CanTravelTo(from, to City) bool {
	for _, route := range gm.routes {
		if route.From == from && route.To == to {
			return true
		}
	}
	return false
}

// GetRoute returns the route between two cities if it exists.
func (gm *GlobalMap) GetRoute(from, to City) (TravelRoute, error) {
	for _, route := range gm.routes {
		if route.From == from && route.To == to {
			return route, nil
		}
	}
	return TravelRoute{}, fmt.Errorf("no route from %s to %s", from, to)
}

// GetConnectedCities returns all cities directly connected to the given city.
func (gm *GlobalMap) GetConnectedCities(from City) []City {
	var connected []City
	for _, route := range gm.routes {
		if route.From == from {
			connected = append(connected, route.To)
		}
	}
	return connected
}

// AllCities returns a list of all cities on the map.
func (gm *GlobalMap) AllCities() []City {
	cities := make([]City, 0, len(gm.cities))
	for city := range gm.cities {
		cities = append(cities, city)
	}
	return cities
}

// String converts City to string for logging and serialization.
func (c City) String() string {
	return string(c)
}
