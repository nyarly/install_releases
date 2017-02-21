package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestResolveReleases(t *testing.T) {
	file, err := os.Open("testdata/releases.json")
	if err != nil {
		t.Fatal(err)
	}

	dec := json.NewDecoder(file)
	rels := []GHRelease{}
	err = dec.Decode(&rels)
	if err != nil {
		t.Fatal(err)
	}

	//log.Printf("%#v\n\n\n", rels)

	found := make(map[string]bool)

	found["0.1.9"] = false

	resolveReleases(rels)

	for _, rel := range rels {
		switch rel.version.String() {
		case "0.1.9":
			found["0.1.9"] = true
			expectedTime := time.Date(2017, 02, 16, 22, 57, 48, 0, time.UTC)

			if !rel.publishTime.Equal(expectedTime) {
				t.Errorf("release 0.1.9 publishTime was %v not %v %#[2]v", rel.publishTime, expectedTime)
			}

			needSuffixes := map[string]int{"": 0, "-0": 0, "-0.1": 0, "-0.1.9": 0}
			for _, suf := range rel.linkSuffixes {
				if _, found := needSuffixes[suf]; found {
					delete(needSuffixes, suf)
				} else {
					t.Errorf("Suffix not expected: %s", suf)
				}
			}
			for suf := range needSuffixes {
				t.Errorf("Suffix not found: %s", suf)
			}
		}
	}

	for ver, f := range found {
		if !f {
			t.Errorf("Didn't find a release with version %s", ver)
		}
	}
}
