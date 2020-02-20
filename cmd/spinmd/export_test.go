package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportCmd(t *testing.T) {
	requests := map[string]int{}
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				requests[fmt.Sprintf("%s %s", r.Method, r.URL.Path)] += 1
				responsePath := fmt.Sprintf("../../test-files/export/responses%s/%s.json", r.URL.Path, r.Method)
				w.Header().Set("Content-Type", "application/json")
				if _, err := os.Stat(responsePath); err != nil {
					responsePath = fmt.Sprintf("../../test-files/export/responses%s/%s.yml", r.URL.Path, r.Method)
					if _, err := os.Stat(responsePath); err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("Content-Type", "application/x-yaml")
				}
				fh, err := os.Open(responsePath)
				require.NoError(t, err)
				defer fh.Close()
				w.WriteHeader(http.StatusOK)
				io.Copy(w, fh)
			},
		),
	)
	defer ts.Close()

	tdir, err := ioutil.TempDir("", "spinnaker-export")
	require.NoError(t, err)
	defer os.RemoveAll(tdir)

	opts := options{
		appName:        "myapp",
		serviceAccount: "myteam@example.com",
		baseURL:        ts.URL,
		configDir:      tdir,
		configFile:     "spinnaker.yml",
	}

	err = exportCmd(&opts, &exportOptions{all: true, envName: "testing"})
	require.NoError(t, err)

	// we expect a bunch of GET requests to variious APIs
	require.Equal(t, map[string]int{
		"GET /applications/myapp/loadBalancers":                       1,
		"GET /applications/myapp/serverGroups":                        1,
		"GET /managed/resources/export/aws/test/cluster/myapp":        1,
		"GET /managed/resources/export/aws/test/security-group/myapp": 1,
		"GET /managed/resources/export/titus/titustest/cluster/myapp": 1,
		"GET /securityGroups/test":                                    1,
		"GET /securityGroups/titustest":                               1,
	}, requests)

	got, err := ioutil.ReadFile(filepath.Join(opts.configDir, opts.configFile))
	require.NoError(t, err)
	expected, err := ioutil.ReadFile("../../test-files/export/spinnaker.yml.expected")
	require.NoError(t, err)
	require.Equal(t, string(expected), string(got))

}
