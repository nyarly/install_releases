package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/samsalisbury/semv"
)

type GHClient struct {
	url               *url.URL
	http, generalHTTP *http.Client
}

type GHRelease struct {
	URL          string `json:"url"`
	Version      string `json:"tag_name"`
	version      semv.Version
	PublishTime  string `json:"published_at"`
	publishTime  time.Time
	Assets       []ReleaseAsset
	linkSuffixes []string
}

type ReleaseAsset struct {
	Name string
	URL  string `json:"browser_download_url"`
}

var versionStrip = regexp.MustCompile(`^\D*`)

func newGHClient(token string) *GHClient {
	github, err := url.Parse("https://api.github.com")
	if err != nil {
		log.Fatal(err)
	}

	hc := &http.Client{}
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		hc = oauth2.NewClient(oauth2.NoContext, ts)
	}

	return &GHClient{
		url:         github,
		http:        hc,
		generalHTTP: &http.Client{},
	}
}

func main() {
	opts := parseOpts()
	log.SetFlags(log.Lshortfile)

	client := newGHClient(os.Getenv("RELEASE_TOKEN"))

	rels := []GHRelease{}
	err := client.fetchJSON(fmt.Sprintf("/repos/%s/releases", opts.githubRepo), &rels)
	if err != nil {
		log.Fatal(err)
	}

	resolveReleases(rels)

	err = os.MkdirAll(opts.store, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	namePattern := regexp.MustCompile(opts.assetPattern)

	wait := &sync.WaitGroup{}
	wait.Add(len(rels))
	for _, rel := range rels {

		go func(rel GHRelease) {
			rel.fetch(client, opts.store, opts.binDir, namePattern)
			wait.Done()
		}(rel)
	}
	log.Printf("waiting...")
	wait.Wait()
	log.Printf("done")
}

func updatePrefixes(prefMap map[string]*GHRelease, format string, rel *GHRelease) {
	verStr := rel.version.Format(format)
	if existing, there := prefMap[verStr]; !there || existing.version.Less(rel.version) {
		prefMap[verStr] = rel
	}
}

func (rel *GHRelease) fetch(cl *GHClient, basePath, linksPath string, pattern *regexp.Regexp) {
	jsonFileName := filepath.Join(basePath, rel.version.String(), "release.json")

	file, err := os.Open(jsonFileName)
	if err == nil {
		defer file.Close()
		dec := json.NewDecoder(file)
		other := &GHRelease{}
		dec.Decode(other)
		if other.URL == rel.URL {
			log.Printf("Skipping download of %s", rel.version.String())
			return
		}
	}
	for _, a := range rel.Assets {
		if pattern.MatchString(a.Name) {
			a.fetch(cl, filepath.Join(basePath, rel.version.String()), linksPath, rel.linkSuffixes)

			file, err := os.Create(jsonFileName)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			enc := json.NewEncoder(file)
			enc.Encode(rel)
			return
		}
	}
}

func (asset *ReleaseAsset) fetch(cl *GHClient, basePath, linksPath string, linkSuffixes []string) {
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	tarReader, err := cl.fetchArchive(asset.URL)
	if err != nil {
		log.Fatal(err)
	}

	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		switch h.Typeflag {
		default:
		case tar.TypeReg, tar.TypeRegA:
			mode := h.FileInfo().Mode()
			parts := strings.Split(h.Name, string(filepath.Separator))[1:]
			newName := filepath.Join(append([]string{basePath}, parts...)...)
			err := os.MkdirAll(filepath.Dir(newName), os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%s -> %s (%o)", h.Name, newName, mode)
			file, err := os.OpenFile(newName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			io.Copy(file, tarReader)

			if (mode & 0100) != 0 {
				for _, suffix := range linkSuffixes {
					exeLink := filepath.Join(append([]string{linksPath}, parts...)...) + suffix
					log.Printf("linking %s to %s", newName, exeLink)
					os.MkdirAll(filepath.Dir(exeLink), os.ModePerm)
					os.Symlink(newName, exeLink)
				}
			}
		}
	}
}

func (cl *GHClient) fetch(path string) (*http.Response, error) {
	url, err := cl.url.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	if url.Host == cl.url.Host {
		return cl.http.Do(req)
	}
	return cl.generalHTTP.Do(req)
}

func (cl *GHClient) fetchJSON(path string, into interface{}) error {
	res, err := cl.fetch(path)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return decode(res.Body, into)
}

func (cl *GHClient) fetchFile(urlPath, filePath string) error {
	res, err := cl.fetch(urlPath)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	size, err := io.Copy(file, res.Body)
	log.Printf("Copied %d bytes", size)
	return err
}

func (cl *GHClient) fetchArchive(urlPath string) (*tar.Reader, error) {
	res, err := cl.fetch(urlPath)
	if err != nil {
		return nil, err
	}
	gunzip, err := gzip.NewReader(res.Body)
	if err != nil {
		return nil, err
	}
	return tar.NewReader(gunzip), nil
}

type loggingReader struct {
	inner io.Reader
}

func (reader *loggingReader) Read(p []byte) (n int, err error) {
	n, err = reader.inner.Read(p)
	//log.Print(n, err, string(p[0:n]))
	return
}

func decode(from io.ReadCloser, to interface{}) error {
	buf := &loggingReader{from}
	dec := json.NewDecoder(buf)
	return dec.Decode(to)
}

func resolveReleases(rels []GHRelease) {
	var err error
	byVersion := make(map[semv.Version]*GHRelease)
	byPrefix := make(map[string]*GHRelease)

	for n := range rels {
		rel := &rels[n]
		rel.linkSuffixes = []string{}
		rel.version, err = semv.Parse(rel.Version)
		if err != nil {
			rel.version = semv.MustParse(versionStrip.ReplaceAllString(rel.Version, ""))

		}
		rel.publishTime, err = time.Parse(time.RFC3339, rel.PublishTime)

		if rel.version.IsPrerelease() {
			continue
		}

		if existing, there := byVersion[rel.version]; !there || existing.publishTime.Before(rel.publishTime) {
			byVersion[rel.version] = rel
			updatePrefixes(byPrefix, "M.m.p", rel)
			updatePrefixes(byPrefix, "M.m", rel)
			updatePrefixes(byPrefix, "M", rel)
			updatePrefixes(byPrefix, "XXX", rel)
		}
	}

	for prefix, rel := range byPrefix {
		if prefix == "XXX" {
			rel.linkSuffixes = append(rel.linkSuffixes, "")
			continue
		}
		rel.linkSuffixes = append(rel.linkSuffixes, "-"+prefix)
	}
}
