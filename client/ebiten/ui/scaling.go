package ui

// Profile describes a device form factor with its logical resolution and DPI expectations.
type Profile struct {
	Name              string
	LogicalWidth      int
	LogicalHeight     int
	MinPhysicalWidth  int
	MaxPhysicalWidth  int
	MinPhysicalHeight int
	MaxPhysicalHeight int
	TargetDPI         float64
	SafeAreaProfile   SafeAreaProfile
}

// SafeAreaProfile defines typical safe-area insets for a device class.
type SafeAreaProfile struct {
	Name        string
	TopInset    int
	BottomInset int
	LeftInset   int
	RightInset  int
}

// StandardProfiles returns a registry of common device profiles.
func StandardProfiles() map[string]Profile {
	return map[string]Profile{
		"phone_portrait": {
			Name:              "phone_portrait",
			LogicalWidth:      720,
			LogicalHeight:     1280,
			MinPhysicalWidth:  360,
			MaxPhysicalWidth:  540,
			MinPhysicalHeight: 720,
			MaxPhysicalHeight: 1920,
			TargetDPI:         326,
			SafeAreaProfile: SafeAreaProfile{
				Name:        "phone_notch",
				TopInset:    44,
				BottomInset: 34,
			},
		},
		"phone_landscape": {
			Name:              "phone_landscape",
			LogicalWidth:      1280,
			LogicalHeight:     720,
			MinPhysicalHeight: 360,
			MaxPhysicalHeight: 540,
			MinPhysicalWidth:  720,
			MaxPhysicalWidth:  1920,
			TargetDPI:         326,
			SafeAreaProfile: SafeAreaProfile{
				Name:       "phone_notch",
				LeftInset:  44,
				RightInset: 44,
			},
		},
		"tablet": {
			Name:              "tablet",
			LogicalWidth:      1024,
			LogicalHeight:     768,
			MinPhysicalWidth:  768,
			MaxPhysicalWidth:  2048,
			MinPhysicalHeight: 576,
			MaxPhysicalHeight: 1536,
			TargetDPI:         264,
			SafeAreaProfile: SafeAreaProfile{
				Name:        "tablet_safe",
				TopInset:    20,
				BottomInset: 20,
				LeftInset:   20,
				RightInset:  20,
			},
		},
		"desktop_16_9": {
			Name:              "desktop_16_9",
			LogicalWidth:      1920,
			LogicalHeight:     1080,
			MinPhysicalWidth:  1280,
			MaxPhysicalWidth:  3840,
			MinPhysicalHeight: 720,
			MaxPhysicalHeight: 2160,
			TargetDPI:         96,
			SafeAreaProfile:   SafeAreaProfile{Name: "desktop"},
		},
		"desktop_ultrawide": {
			Name:              "desktop_ultrawide",
			LogicalWidth:      3440,
			LogicalHeight:     1440,
			MinPhysicalWidth:  2560,
			MaxPhysicalWidth:  5120,
			MinPhysicalHeight: 1080,
			MaxPhysicalHeight: 2160,
			TargetDPI:         96,
			SafeAreaProfile:   SafeAreaProfile{Name: "desktop"},
		},
	}
}

// ResolveProfile selects the best-matching profile for the given physical dimensions.
func ResolveProfile(physicalWidth, physicalHeight int) Profile {
	profiles := StandardProfiles()
	isPortrait := physicalHeight > physicalWidth

	type scored struct {
		name  string
		score float64
	}
	var candidates []scored

	for name, p := range profiles {
		if physicalWidth < p.MinPhysicalWidth || physicalWidth > p.MaxPhysicalWidth {
			continue
		}
		if physicalHeight < p.MinPhysicalHeight || physicalHeight > p.MaxPhysicalHeight {
			continue
		}

		orientationMatch := 1.0
		profilePortrait := p.LogicalHeight > p.LogicalWidth
		if profilePortrait != isPortrait {
			orientationMatch = 0.5
		}

		widthDist := float64(physicalWidth - p.MinPhysicalWidth)
		heightDist := float64(physicalHeight - p.MinPhysicalHeight)
		distance := (widthDist*widthDist + heightDist*heightDist)
		score := orientationMatch / (1.0 + distance*0.0001)

		candidates = append(candidates, scored{name, score})
	}

	if len(candidates) == 0 {
		return profiles["desktop_16_9"]
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return profiles[best.name]
}

// ScaleFactor computes the device pixel ratio for the given physical and logical dimensions.
func ScaleFactor(physicalWidth, logicalWidth float64) float64 {
	if logicalWidth <= 0 {
		return 1.0
	}
	scale := physicalWidth / logicalWidth
	if scale < 0.5 {
		scale = 0.5
	}
	if scale > 4.0 {
		scale = 4.0
	}
	return scale
}

// TextScaleForProfile returns a multiplier for font size based on profile type.
func TextScaleForProfile(profile Profile) float64 {
	switch profile.Name {
	case "phone_portrait", "phone_landscape":
		return 1.0
	case "tablet":
		return 1.2
	case "desktop_16_9":
		return 1.4
	case "desktop_ultrawide":
		return 1.6
	default:
		return 1.0
	}
}

// IconScaleForProfile returns a multiplier for icon/sprite size based on profile type.
func IconScaleForProfile(profile Profile) float64 {
	return TextScaleForProfile(profile)
}
