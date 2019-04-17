package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestCullReleases(t *testing.T) {
	rels := loadTestReleases(t)
	resolveReleases(rels)

	fetched := map[string]bool{}

	for _, rel := range rels {
		fetched[rel.version.String()] = true
	}

	culled := cullReleases(rels, "2.10.3")

	found := map[string]bool{}

	for _, rel := range culled {
		found[rel.version.String()] = true
	}

	expectVersion := func(v string) {
		if !fetched[v] {
			t.Errorf("Version %q was expected but not in test set!", v)
			return
		}
		if !found[v] {
			t.Errorf("Version %q was expected but missing", v)
		}
	}

	expectCulled := func(v string) {
		if !fetched[v] {
			t.Errorf("Version %q was expected but not in test set!", v)
			return
		}
		if found[v] {
			t.Errorf("Version %q was expected to be culled, but present!", v)
		}
	}

	expectVersion("0.5.2")
	expectVersion("0.1.9")
	expectCulled("0.1.4")
	expectCulled("0.1")
}

func TestResolveReleases(t *testing.T) {
	rels := loadTestReleases(t)
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

func loadTestReleases(t *testing.T) []GHRelease {
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

	return rels
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
