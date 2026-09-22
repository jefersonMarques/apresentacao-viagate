package catalog

import "testing"

func TestCargoScorePricingModelsKeepItemAndBundleOptionsSeparated(t *testing.T) {
	var item Item
	var bundle Item
	for _, group := range Groups {
		if group.ID != "score" {
			continue
		}
		for _, candidate := range group.Items {
			switch candidate.ID {
			case "score-item-driver-register":
				item = candidate
			case "score-bundle-register":
				bundle = candidate
			}
		}
	}

	if item.ID == "" || bundle.ID == "" {
		t.Fatal("expected score item and bundle fixtures")
	}
	if !ModelAllows(item, "per_item") || ModelAllows(bundle, "per_item") {
		t.Fatal("per_item must allow only item-based score options")
	}
	if ModelAllows(item, "bundle") || !ModelAllows(bundle, "bundle") {
		t.Fatal("bundle must allow only bundle-based score options")
	}
	if !ModelAllows(item, "item_and_bundle") || !ModelAllows(bundle, "item_and_bundle") {
		t.Fatal("item_and_bundle must allow both score option families")
	}
}
