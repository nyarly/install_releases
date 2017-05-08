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
	found["0.5.2"] = false

	resolveReleases(rels)

	gotSuffixes := map[string]int{}

	for _, rel := range rels {
		for _, suf := range rel.linkSuffixes {
			if _, has := gotSuffixes[suf]; !has {
				gotSuffixes[suf] = 0
			}
			gotSuffixes[suf]++
		}

		switch rel.version.String() {
		case "0.5.2":
			found["0.5.2"] = true
			expectSuffixes(t, rel, "", "-0", "-0.5", "-0.5.2")
		case "0.1.9":
			found["0.1.9"] = true
			expectedTime := time.Date(2017, 02, 16, 22, 57, 48, 0, time.UTC)

			if !rel.publishTime.Equal(expectedTime) {
				t.Errorf("release 0.1.9 publishTime was %v not %v %#[2]v", rel.publishTime, expectedTime)
			}

			expectSuffixes(t, rel, "-0.1", "-0.1.9")
		}
	}

	for suf, count := range gotSuffixes {
		if count != 1 {
			t.Errorf("%d releases think they own the %q suffix.", count, suf)
		}
	}

	for ver, f := range found {
		if !f {
			t.Errorf("Didn't find a release with version %s", ver)
		}
	}
}

func expectSuffixes(t *testing.T, rel GHRelease, sufs ...string) {
	needSuffixes := map[string]int{}
	for _, suf := range sufs {
		needSuffixes[suf] = 0
	}
	for _, suf := range rel.linkSuffixes {
		if _, found := needSuffixes[suf]; found {
			delete(needSuffixes, suf)
		} else {
			t.Errorf("Release %q: suffix not expected: %q", rel.Version, suf)
		}
	}
	for suf := range needSuffixes {
		t.Errorf("Release %q: suffix not found: %q", rel.Version, suf)
	}
}
