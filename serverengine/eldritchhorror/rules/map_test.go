package rules

import "testing"

func TestNewGlobalMap(t *testing.T) {
	gm := NewGlobalMap()

	// Verify minimum number of cities (20 in the initial map)
	allCities := gm.AllCities()
	if len(allCities) < 18 {
		t.Errorf("expected at least 18 cities, got %d", len(allCities))
	}
}

func TestIsValidCity(t *testing.T) {
	gm := NewGlobalMap()

	validCities := []City{
		CityArkham, CitySanFrancisco, CityBuenosAires,
		CityLondon, CityRome, CityIstanbul,
		CityTokyo, CityShanghai, CityBangkok,
		CityCairo, CityTunis, CityCapeTown,
		CitySydney, CityHonolulu,
	}

	for _, city := range validCities {
		if !gm.IsValidCity(city) {
			t.Errorf("expected %s to be valid", city)
		}
	}

	// Test invalid city
	if gm.IsValidCity("invalidCity") {
		t.Error("expected invalidCity to be invalid")
	}
}

func TestCanTravelTo(t *testing.T) {
	gm := NewGlobalMap()

	// Test valid routes
	validRoutes := []struct {
		from City
		to   City
	}{
		{CityArkham, CitySanFrancisco},
		{CitySanFrancisco, CityArkham}, // Bidirectional
		{CityLondon, CityRome},
		{CityRome, CityLondon}, // Bidirectional
	}

	for _, test := range validRoutes {
		if !gm.CanTravelTo(test.from, test.to) {
			t.Errorf("expected route from %s to %s", test.from, test.to)
		}
	}

	// Test invalid route (cities not directly connected)
	if gm.CanTravelTo(CityArkham, CityTokyo) {
		t.Error("expected no direct route from Arkham to Tokyo")
	}
}

func TestGetRoute(t *testing.T) {
	gm := NewGlobalMap()

	// Test valid route retrieval
	route, err := gm.GetRoute(CityArkham, CitySanFrancisco)
	if err != nil {
		t.Fatalf("expected route from Arkham to San Francisco, got error: %v", err)
	}
	if route.From != CityArkham || route.To != CitySanFrancisco {
		t.Errorf("unexpected route: %+v", route)
	}
	if route.Cost <= 0 {
		t.Error("expected positive travel cost")
	}
	if route.Type == "" {
		t.Error("expected route type to be set")
	}

	// Test invalid route
	_, err = gm.GetRoute(CityArkham, CityTokyo)
	if err == nil {
		t.Error("expected error for non-existent route")
	}
}

func TestGetConnectedCities(t *testing.T) {
	gm := NewGlobalMap()

	connected := gm.GetConnectedCities(CityArkham)
	if len(connected) == 0 {
		t.Error("expected Arkham to have connected cities")
	}

	// Verify San Francisco is connected to Arkham
	found := false
	for _, city := range connected {
		if city == CitySanFrancisco {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected San Francisco to be connected to Arkham")
	}
}

func TestCityString(t *testing.T) {
	if CityArkham.String() != "arkham" {
		t.Errorf("expected 'arkham', got %s", CityArkham.String())
	}
	if CityLondon.String() != "london" {
		t.Errorf("expected 'london', got %s", CityLondon.String())
	}
}

func TestRouteTypes(t *testing.T) {
	gm := NewGlobalMap()

	// Test that different route types exist
	routeTypes := make(map[RouteType]bool)
	for _, route := range gm.routes {
		routeTypes[route.Type] = true
	}

	expectedTypes := []RouteType{RouteTypeLand, RouteTypeShip, RouteTypeTrain}
	for _, rt := range expectedTypes {
		if !routeTypes[rt] {
			t.Errorf("expected route type %s to exist in map", rt)
		}
	}
}

func TestBidirectionalRoutes(t *testing.T) {
	gm := NewGlobalMap()

	// Verify that all routes are bidirectional
	for _, route := range gm.routes {
		reverseRoute, err := gm.GetRoute(route.To, route.From)
		if err != nil {
			t.Errorf("expected reverse route from %s to %s", route.To, route.From)
			continue
		}
		if reverseRoute.Cost != route.Cost {
			t.Errorf("expected same cost for bidirectional route, got %d and %d",
				route.Cost, reverseRoute.Cost)
		}
		if reverseRoute.Type != route.Type {
			t.Errorf("expected same type for bidirectional route, got %s and %s",
				route.Type, reverseRoute.Type)
		}
	}
}

func TestTravelCosts(t *testing.T) {
	gm := NewGlobalMap()

	// Verify all routes have reasonable costs (1-3 actions)
	for _, route := range gm.routes {
		if route.Cost < 1 || route.Cost > 3 {
			t.Errorf("unexpected travel cost %d for route %s -> %s",
				route.Cost, route.From, route.To)
		}
	}
}
