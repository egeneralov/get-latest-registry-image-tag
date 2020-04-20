package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Masterminds/semver"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	url          string
	answer_token AnswerToken
	tag_list     TagList
	repository   = "cockroachdb/cockroach"
	prefix       string
	headers      = make(map[string][]string)
	registry     = "https://registry-1.docker.io"
	vs           []*semver.Version
)

func main() {
	// 	flag.StringVar(&registry, "registry", "https://registry-1.docker.io", "registry url")
	flag.StringVar(&repository, "repository", "cockroachdb/cockroach", "[library/docker | username/repository]")
	flag.Parse()

	// curl --user 'username:password' 'https://gitlab.domain.com/jwt/auth?client_id=docker&offline_token=true&service=container_registry&scope=repository:your-repo-name:push,pull'

	//   if registry == "https://registry-1.docker.io" {
	url = fmt.Sprintf(
		"https://auth.docker.io/token?service=registry.docker.io&scope=repository:%v:pull",
		repository,
	)

	tokenBodyBytes, err := Get(url, headers)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(tokenBodyBytes, &answer_token)
	if err != nil {
		panic(err)
	}

	headers["Authorization"] = []string{
		fmt.Sprintf(
			"Bearer %v", answer_token.Token,
		),
	}

	//   }

	url = fmt.Sprintf(
		"%v/v2/%v/tags/list",
		registry, repository,
	)

	TagListBytes, err := Get(url, headers)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(TagListBytes, &tag_list)
	if err != nil {
		panic(err)
	}

	// 	re := regexp.MustCompile(`v?\d+\.\d+\.?\d+?\.?`)
	re := regexp.MustCompile(`^v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?$`)

	tags := []string{}

	for _, tag := range tag_list.Tags {
		if re.MatchString(tag) {
			new_tag := strings.Replace(tag, "v", "", 1)
			if new_tag != tag {
				prefix = "v"
			}
			tags = append(tags, new_tag)
		}
	}

	for _, r := range tags {
		v, err := semver.NewVersion(r)
		if err != nil {
			continue
		}
		vs = append(vs, v)
	}

	sort.Sort(semver.Collection(vs))

	if len(vs) > 0 {
		fmt.Printf(
			"%v%v",
			prefix,
			vs[len(vs)-1],
		)
	} else {
		if len(tags) == 0 {
			fmt.Printf("latest")
		} else {
			fmt.Printf(tags[0])
		}
	}
}

func Get(url string, headers map[string][]string) ([]byte, error) {
	c := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, err
	}
	if headers != nil {
		r.Header = http.Header(headers)
	}
	res, err := c.Do(r)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()
	lr := &io.LimitedReader{res.Body, 1000000}
	rb, err := ioutil.ReadAll(lr)
	if err != nil {
		return []byte{}, err
	}
	return rb, nil
}

type TagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type AnswerToken struct {
	Token       string    `json:"token"`
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	IssuedAt    time.Time `json:"issued_at"`
}
